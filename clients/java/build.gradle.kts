plugins {
    java
    `java-library`
    `maven-publish`
    signing
}

// Read version from root VERSION file
val vibiumVersion = file("../../VERSION").readText().trim()
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
val validateCapabilityMarkers by tasks.registering {
    inputs.dir("src/test/java/com/vibium/engine")
    inputs.file("../../tests/capabilities.json")
    doLast {
        val manifestText = file("../../tests/capabilities.json").readText()
        val manifest = Regex("\"([^\"]+)\"\\s*:\\s*\\[([^]]*)]")
            .findAll(manifestText)
            .associate { match ->
                match.groupValues[1] to Regex("\"([^\"]+)\"")
                    .findAll(match.groupValues[2]).map { it.groupValues[1] }.toSet()
            }
        fileTree("src/test/java/com/vibium/engine") {
            include("*Test.java")
        }.forEach { file ->
            val source = file.readText()
            if (source.contains("@Test") && !source.contains("@RequiresCapability(")) {
                throw GradleException("${file.path}: unmarked test in Java cross-engine root")
            }
            Regex("@RequiresCapability\\(([^)]*)\\)").findAll(source).forEach { marker ->
                Regex("\"([^\"]+)\"").findAll(marker.groupValues[1]).forEach { nameMatch ->
                    val name = nameMatch.groupValues[1]
                    val engines = manifest[name]
                        ?: throw GradleException("${file.path}: unknown capability $name")
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

// Don't fail build if native binaries aren't present (dev mode)
tasks.named("processResources") {
    dependsOn(tasks.named("copyNativeBinaries"))
}

// sourcesJar also reads src/main/resources, so it needs the same dependency
tasks.named("sourcesJar") {
    dependsOn(tasks.named("copyNativeBinaries"))
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
    }
}

signing {
    useGpgCmd()
    sign(publishing.publications["mavenJava"])
}

// Only sign when publishing
tasks.withType<Sign>().configureEach {
    onlyIf { gradle.taskGraph.hasTask(":publish") }
}
