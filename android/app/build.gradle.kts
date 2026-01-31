import org.gradle.testing.jacoco.plugins.JacocoTaskExtension
import java.io.FileInputStream
import java.util.Properties

// ==================== PLUGINS ====================

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.android)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.serialization)
    alias(libs.plugins.spotless)
    alias(libs.plugins.detekt)
    id("jacoco")
}

// ==================== ANDROID CONFIG ====================
val localPropsFile = rootProject.file("config.properties")
val localProps = Properties()
if (localPropsFile.exists()) {
    FileInputStream(localPropsFile).use { localProps.load(it) }
}
for (k in localProps.stringPropertyNames()) {
    project.extensions.extraProperties[k] = localProps.getProperty(k)
}


android {
    namespace = "com.fleetdm.agent"
    compileSdk = 36

    defaultConfig {
        applicationId = "com.fleetdm.agent"
        minSdk = 33
        targetSdk = 36
        versionCode = 6
        versionName = "1.1.0"

        buildConfigField("String", "INFO_URL", "\"https://fleetdm.com/better\"")

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"

        // Pass integration test flag from project property to instrumentation runner
        if (project.hasProperty("runIntegrationTests")) {
            testInstrumentationRunnerArguments["runIntegrationTests"] = "true"
        }
    }

    testOptions {
        unitTests {
            isIncludeAndroidResources = false
            all {
                it.apply {
                    // Validate integration test configuration
                    if (project.hasProperty("runIntegrationTests")) {
                        // Check for required SCEP configuration
                        if (!project.hasProperty("scep.url") || !project.hasProperty("scep.challenge")) {
                            throw GradleException(
                                """
                                |
                                |ERROR: Integration tests require SCEP server configuration.
                                |
                                |Please provide both required properties:
                                |  -Pscep.url=<SCEP_SERVER_URL>
                                |  -Pscep.challenge=<SCEP_CHALLENGE>
                                |
                                |Example:
                                |  ./gradlew test -PrunIntegrationTests=true \
                                |    -Pscep.url=https://your-scep-server.com/scep \
                                |    -Pscep.challenge=your-challenge-password
                                |
                                """.trimMargin(),
                            )
                        }

                        systemProperty("runIntegrationTests", "true")
                        systemProperty("scep.url", project.property("scep.url").toString())
                        systemProperty("scep.challenge", project.property("scep.challenge").toString())
                    }

                    // Enable jacoco coverage for Robolectric tests
                    extensions.configure<JacocoTaskExtension> {
                        isIncludeNoLocationClasses = true
                        excludes = listOf("jdk.internal.*")
                    }
                }
            }
        }
    }

    // Load keystore properties for release signing
    val keystorePropertiesFile = rootProject.file("keystore.properties")
    val keystoreProperties = Properties()
    if (keystorePropertiesFile.exists()) {
        FileInputStream(keystorePropertiesFile).use { keystoreProperties.load(it) }
    }

    signingConfigs {
        create("release") {
            if (keystorePropertiesFile.exists()) {
                storeFile = rootProject.file(keystoreProperties.getProperty("storeFile"))
                storePassword = keystoreProperties.getProperty("storePassword")
                keyAlias = keystoreProperties.getProperty("keyAlias")
                keyPassword = keystoreProperties.getProperty("keyPassword")
            }
        }
    }

    buildTypes {
        debug {
            enableUnitTestCoverage = true
            enableAndroidTestCoverage = true
            buildConfigField("String", "FLEET_BASE_URL", "\"${project.findProperty("FLEET_BASE_URL") ?: ""}\"")
            buildConfigField("String", "FLEET_NODE_KEY", "\"${project.findProperty("FLEET_NODE_KEY") ?: ""}\"")
            buildConfigField("String", "FLEET_ENROLL_SECRET", "\"${project.findProperty("FLEET_ENROLL_SECRET") ?: ""}\"")
            buildConfigField("boolean", "FLEET_ALLOW_INSECURE_TLS", "true")
        }
        release {
            buildConfigField("boolean", "FLEET_ALLOW_INSECURE_TLS", "false")

            if (keystorePropertiesFile.exists()) {
                signingConfig = signingConfigs.getByName("release")
            }
            isMinifyEnabled = false
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro",
            )
        }
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_11
        targetCompatibility = JavaVersion.VERSION_11
    }
    buildFeatures {
        compose = true
        buildConfig = true
    }
    packaging {
        resources {
            excludes +=
                setOf(
                    "META-INF/DEPENDENCIES",
                    "META-INF/DEPENDENCIES.txt",
                    "META-INF/LICENSE",
                    "META-INF/LICENSE.txt",
                    "META-INF/LICENSE.md",
                    "META-INF/LICENSE-notice.md",
                    "META-INF/NOTICE",
                    "META-INF/NOTICE.txt",
                    "META-INF/NOTICE.md",
                    "META-INF/notice.txt",
                    "META-INF/license.txt",
                    "META-INF/license.md",
                    "META-INF/dependencies.txt",
                    "META-INF/LGPL2.1",
                    "META-INF/AL2.0",
                    "META-INF/LGPL3.0",
                    "META-INF/*.kotlin_module",
                )
        }
    }
}

// ==================== KOTLIN & JAVA TOOLCHAIN ====================

kotlin {
    jvmToolchain(17)
    compilerOptions {
        jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_11)
        freeCompilerArgs.add("-opt-in=androidx.compose.material3.ExperimentalMaterial3Api")
    }
}

java {
    toolchain {
        languageVersion.set(JavaLanguageVersion.of(17))
    }
}

// ==================== CODE QUALITY ====================

detekt {
    buildUponDefaultConfig = true
    allRules = false
    config.setFrom("$projectDir/detekt.yml")
}

// Don't run Detekt automatically in local builds, only in CI
tasks.named("check") {
    setDependsOn(dependsOn.filterNot { it.toString().contains("detekt") })
}

// ==================== JACOCO COVERAGE ====================

jacoco {
    toolVersion = "0.8.14"
}

tasks.register<JacocoReport>("jacocoTestReport") {
    dependsOn("testDebugUnitTest")

    reports {
        xml.required.set(true)
        html.required.set(true)
    }

    val fileFilter = listOf(
        "**/R.class",
        "**/R\$*.class",
        "**/BuildConfig.*",
        "**/Manifest*.*",
        "**/*Test*.*",
        "android/**/*.*",
        "**/compose/**/*.*",
    )

    val debugTree = fileTree("${layout.buildDirectory.get()}/tmp/kotlin-classes/debug") {
        exclude(fileFilter)
    }

    val mainSrc = listOf(
        "${project.projectDir}/src/main/java",
        "${project.projectDir}/src/main/kotlin",
    )

    sourceDirectories.setFrom(files(mainSrc))
    classDirectories.setFrom(files(debugTree))
    executionData.setFrom(
        fileTree(layout.buildDirectory) {
            include(
                // Unit test coverage
                "outputs/unit_test_code_coverage/debugUnitTest/testDebugUnitTest.exec",
                // Instrumented test coverage
                "outputs/code_coverage/debugAndroidTest/connected/**/*.ec",
            )
        },
    )
}

// ==================== SPOTLESS FORMATTING ====================

spotless {
    kotlin {
        target("**/*.kt")
        targetExclude("**/build/**/*.kt")
        ktlint()
    }
    kotlinGradle {
        target("*.gradle.kts")
        ktlint()
    }
}

// ==================== DEPENDENCIES ====================

dependencies {
    // AndroidX and Compose
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.lifecycle.runtime.ktx)
    implementation(libs.androidx.activity.compose)
    implementation(platform(libs.androidx.compose.bom))
    implementation(libs.androidx.compose.ui)
    implementation(libs.androidx.compose.ui.graphics)
    implementation(libs.androidx.compose.ui.tooling.preview)
    implementation(libs.androidx.compose.material3)
    implementation(libs.androidx.compose.material.icons.extended)
    implementation(libs.androidx.datastore.preferences)
    implementation(libs.androidx.navigation.compose)
    implementation(libs.androidx.work.runtime.ktx)
    implementation(libs.amapi.sdk)
    implementation(libs.kotlinx.serialization.json)

    implementation("com.squareup.okhttp3:okhttp:4.12.0")


    // SCEP (Simple Certificate Enrollment Protocol)
    implementation(libs.jscep)

    // Bouncy Castle - Cryptography provider
    implementation(libs.bouncycastle.bcprov)
    implementation(libs.bouncycastle.bcpkix)

    // Apache Commons - Utilities used by jScep
    implementation(libs.commons.codec)

    // Logging - Required by jScep
    implementation(libs.slf4j.api)
    implementation(libs.slf4j.simple)

    // Testing
    testImplementation(libs.junit)
    testImplementation(libs.androidx.work.testing)
    testImplementation(libs.robolectric)
    testImplementation(libs.kotlinx.coroutines.test)
    testImplementation(libs.json) // For JSON parsing in unit tests
    testImplementation(libs.okhttp.mockwebserver)

    androidTestImplementation(libs.androidx.junit)
    androidTestImplementation(libs.androidx.espresso.core)
    androidTestImplementation(platform(libs.androidx.compose.bom))
    androidTestImplementation(libs.androidx.compose.ui.test.junit4)
    androidTestImplementation(libs.mockk.android)

    debugImplementation(libs.androidx.compose.ui.tooling)
    debugImplementation(libs.androidx.compose.ui.test.manifest)
}
