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
    useJUnitPlatform()
    // Pass VIBIUM_BIN_PATH to tests if set
    environment("VIBIUM_BIN_PATH", System.getenv("VIBIUM_BIN_PATH") ?: "")
    // Each fork is a JVM with its own Chrome (~16s/launch on macOS). Default
    // 4 mirrors test-js's JS_PARALLEL and test-python's PY_PARALLEL. Override
    // with -PjavaParallel=N (Makefile: JAVA_PARALLEL).
    maxParallelForks = (project.findProperty("javaParallel") as String?)?.toIntOrNull() ?: 4
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
