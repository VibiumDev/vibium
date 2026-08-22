plugins {
    java
    `java-library`
    `maven-publish`
    signing
}

// Read version from root VERSION file
val vibiumVersion = providers.gradleProperty("vibiumVersion")
    .orElse(file("../../VERSION").readText().trim())
    .get()
version = vibiumVersion
group = "com.vibium"

java {
    sourceCompatibility = JavaVersion.VERSION_11
    targetCompatibility = JavaVersion.VERSION_11
    withSourcesJar()
    withJavadocJar()
}

repositories {
    mavenCentral()
}

dependencies {
    implementation("com.google.code.gson:gson:2.11.0")

    testImplementation("org.junit.jupiter:junit-jupiter:5.11.3")
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")
}

tasks.test {
    useJUnitPlatform {
        if (project.hasProperty("capabilityOnly")) {
            includeTags("cross-engine")
        }
    }
    // Integration tests drive the vibium binary and a live browser — neither is
    // a Gradle input, so never let cached results skip the run.
    outputs.upToDateWhen { false }
    // Pass VIBIUM_BIN_PATH to tests if set
    environment("VIBIUM_BIN_PATH", System.getenv("VIBIUM_BIN_PATH") ?: "")
    // Each fork is a JVM with its own Chrome (~16s/launch on macOS). Default
    // 4 mirrors test-js's JS_PARALLEL and test-python's PY_PARALLEL. Override
    // with -PjavaParallel=N (Makefile: JAVA_PARALLEL).
    maxParallelForks = (project.findProperty("javaParallel") as String?)?.toIntOrNull() ?: 4
}

// Cross-engine browser tests live in a physical root so missing capability
// declarations cannot turn into a new filename allowlist. Validate the source
// before JUnit launches a browser; method-level markers add requirements to the
// mandatory class-level baseline.
// -PcapabilityMarkerRoot points the validator at a synthetic tree so the
// fixture runner can assert it still rejects bad markers.
val capabilityMarkerRoot =
    (findProperty("capabilityMarkerRoot") as String?) ?: "src/test/java/com/vibium/engine"

val validateCapabilityMarkers by tasks.registering {
    inputs.dir(capabilityMarkerRoot)
    inputs.file("../../tests/capabilities.json")
    doLast {
        val manifestText = file("../../tests/capabilities.json").readText()
        val manifest = Regex("\"([^\"]+)\"\\s*:\\s*\\[([^]]*)]")
            .findAll(manifestText)
            .associate { match ->
                match.groupValues[1] to Regex("\"([^\"]+)\"")
                    .findAll(match.groupValues[2]).map { it.groupValues[1] }.toSet()
            }
        // The marker must sit on the class declaration: -PcapabilityOnly selects
        // on the class-level tag, so a file with only method-level markers would
        // silently drop its unmarked methods from the engine run.
        val classMarker = Regex(
            "@RequiresCapability\\([^)]*\\)\\s*(?:@[\\w.]+(?:\\([^)]*\\))?\\s*)*" +
            "(?:\\b(?:public|final|abstract)\\b\\s+)*class\\s"
        )
        fileTree(capabilityMarkerRoot) {
            include("**/*Test.java")
        }.forEach { file ->
            val source = file.readText()
            if (source.contains("@Test") && !classMarker.containsMatchIn(source)) {
                throw GradleException(
                    "${file.path}: missing class-level @RequiresCapability " +
                    "(method-level markers only add to the class baseline)"
                )
            }
            Regex("@RequiresCapability\\(([^)]*)\\)").findAll(source).forEach { marker ->
                Regex("\"([^\"]+)\"").findAll(marker.groupValues[1]).forEach { nameMatch ->
                    val name = nameMatch.groupValues[1]
                    val engines = manifest[name]
                        ?: throw GradleException("${file.path}: unknown capability $name")
                    // The manifest must not list an engine for a capability
                    // unless chrome is also listed; empty entries are fine.
                    // Add an exemption mechanism before introducing one.
                    if (System.getenv("VIBIUM_CAPABILITY_AUDIT") == "1" &&
                        engines.isNotEmpty() && "chrome" !in engines) {
                        throw GradleException("${file.path}: Chrome audit rejected skip for $name")
                    }
                }
            }
        }
    }
}

tasks.test {
    dependsOn(validateCapabilityMarkers)
    environment("VIBIUM_CAPABILITIES_FILE", file("../../tests/capabilities.json").absolutePath)
    environment("VIBIUM_CAPABILITY_AUDIT", System.getenv("VIBIUM_CAPABILITY_AUDIT") ?: "")
}

// Copy native binaries into resources for JAR packaging
tasks.register<Copy>("copyNativeBinaries") {
    from("../../clicker/bin") {
        include("vibium-darwin-amd64")
        include("vibium-darwin-arm64")
        include("vibium-linux-amd64")
        include("vibium-linux-arm64")
        include("vibium-windows-amd64.exe")
    }
    into("src/main/resources/natives")
}

// BinaryResolver keys its extraction cache on this resource; without it every
// jar extracts to vibium-unknown and the first install pins its binary (#330).
val writeVersionResource by tasks.registering {
    val versionFile = file("src/main/resources/vibium-version.txt")
    inputs.property("version", vibiumVersion)
    outputs.file(versionFile)
    doLast {
        versionFile.parentFile.mkdirs()
        versionFile.writeText(vibiumVersion + "\n")
    }
}

// Don't fail build if native binaries aren't present (dev mode)
tasks.named("processResources") {
    dependsOn(tasks.named("copyNativeBinaries"), writeVersionResource)
}

// copyNativeBinaries is a Copy over ../../clicker/bin; with that directory
// unpopulated it succeeds having copied nothing, and the JAR ships with no
// binaries at all. Dev builds tolerate that deliberately. A publish must not:
// Maven Central releases are immutable, so an empty JAR can only be superseded.
val requiredNatives = listOf(
    "vibium-darwin-amd64",
    "vibium-darwin-arm64",
    "vibium-linux-amd64",
    "vibium-linux-arm64",
    "vibium-windows-amd64.exe",
)
val verifyNativeBinaries by tasks.registering {
    dependsOn(tasks.named("copyNativeBinaries"))
    doLast {
        val dir = file("src/main/resources/natives")
        val missing = requiredNatives.filter { !dir.resolve(it).exists() }
        if (missing.isNotEmpty()) {
            throw GradleException(
                "refusing to publish a JAR without native binaries: " +
                    missing.joinToString(", ") +
                    "\nRun 'make build-go-all' from the repo root, then stage again."
            )
        }
    }
}

tasks.withType<org.gradle.api.publish.maven.tasks.PublishToMavenRepository>()
    .configureEach { dependsOn(verifyNativeBinaries) }

// sourcesJar also reads src/main/resources, so it needs the same dependency
tasks.named("sourcesJar") {
    dependsOn(tasks.named("copyNativeBinaries"), writeVersionResource)
}

// Copy runtime dependencies for JShell / standalone use
tasks.register<Copy>("copyDependencies") {
    from(configurations.runtimeClasspath)
    into("build/dependencies")
}
tasks.named("build") { dependsOn("copyDependencies") }

tasks.named<Jar>("jar") {
    manifest {
        attributes("Main-Class" to "com.vibium.CLI")
    }
}

tasks.named<Javadoc>("javadoc") {
    (options as StandardJavadocDocletOptions).apply {
        addStringOption("Xdoclint:none", "-quiet")
    }
}

publishing {
    publications {
        create<MavenPublication>("mavenJava") {
            from(components["java"])

            pom {
                name.set("Vibium")
                description.set("Browser automation for AI agents and humans")
                url.set("https://github.com/VibiumDev/vibium")

                licenses {
                    license {
                        name.set("The Apache License, Version 2.0")
                        url.set("https://www.apache.org/licenses/LICENSE-2.0.txt")
                    }
                }

                developers {
                    developer {
                        id.set("vibium")
                        name.set("Vibium")
                        email.set("hello@vibium.com")
                    }
                }

                scm {
                    connection.set("scm:git:git://github.com/VibiumDev/vibium.git")
                    developerConnection.set("scm:git:ssh://github.com/VibiumDev/vibium.git")
                    url.set("https://github.com/VibiumDev/vibium")
                }
            }
        }
    }

    repositories {
        maven {
            name = "staging"
            url = uri(layout.buildDirectory.dir("staging-deploy"))
        }
        maven {
            name = "centralSnapshots"
            url = uri("https://central.sonatype.com/repository/maven-snapshots/")
            credentials {
                username = System.getenv("MAVEN_CENTRAL_USERNAME")
                password = System.getenv("MAVEN_CENTRAL_PASSWORD")
            }
        }
    }
}

signing {
    useGpgCmd()
    sign(publishing.publications["mavenJava"])
}

// Sign for any publish to a real repository. Keying on ":publish" matched
// only the aggregate task, so every publishAllPublicationsTo<Name>Repository
// invocation skipped signing silently -- the manual release staged unsigned
// artifacts, and the nightly pushed unsigned snapshots despite importing a
// key and passing a passphrase. publishToMavenLocal stays unsigned.
tasks.withType<Sign>().configureEach {
    onlyIf {
        gradle.taskGraph.allTasks.any {
            it is org.gradle.api.publish.maven.tasks.PublishToMavenRepository
        }
    }
}
