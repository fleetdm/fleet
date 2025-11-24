plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.android)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.spotless)
    alias(libs.plugins.detekt)
}

android {
    namespace = "com.fleetdm.agent"
    compileSdk = 36

    defaultConfig {
        applicationId = "com.fleetdm.agent"
        minSdk = 33
        targetSdk = 36
        versionCode = 1
        versionName = "1.0"

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
                    // Pass integration test flag and SCEP configuration to unit tests
                    if (project.hasProperty("runIntegrationTests")) {
                        systemProperty("runIntegrationTests", "true")
                    }
                    if (project.hasProperty("scep.url")) {
                        systemProperty("scep.url", project.property("scep.url").toString())
                    }
                    if (project.hasProperty("scep.challenge")) {
                        systemProperty("scep.challenge", project.property("scep.challenge").toString())
                    }
                }
            }
        }
    }

    buildTypes {
        release {
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
    }
    packaging {
        resources {
            excludes += setOf(
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
                "META-INF/*.kotlin_module"
            )
        }
    }
}

kotlin {
    jvmToolchain(17)
    compilerOptions {
        jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_11)
    }
}

java {
    toolchain {
        languageVersion.set(JavaLanguageVersion.of(17))
    }
}

detekt {
    buildUponDefaultConfig = true
    allRules = false
    config.setFrom("$projectDir/detekt.yml")
}

// Don't run Detekt automatically in local builds, only in CI
tasks.named("check") {
    setDependsOn(dependsOn.filterNot { it.toString().contains("detekt") })
}

spotless {
    kotlin {
        target("**/*.kt")
        targetExclude("**/build/**/*.kt")
        ktlint().editorConfigOverride(
            mapOf(
                // Jetpack Compose requires Composable functions to start with uppercase (PascalCase)
                "ktlint_standard_function-naming" to "disabled",
                // Android conventionally uses uppercase TAG constants for logging
                "ktlint_standard_property-naming" to "disabled",
            ),
        )
    }
    kotlinGradle {
        target("*.gradle.kts")
        ktlint()
    }
}

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

    // SCEP (Simple Certificate Enrollment Protocol)
    implementation("com.google.code.jscep:jscep:3.0.1")

    // Bouncy Castle - Cryptography provider
    implementation("org.bouncycastle:bcprov-jdk18on:1.78.1")
    implementation("org.bouncycastle:bcpkix-jdk18on:1.78.1")

    // Apache Commons - Utilities used by jScep
    implementation("commons-codec:commons-codec:1.17.1")

    // Logging - Required by jScep
    implementation("org.slf4j:slf4j-api:2.0.16")
    implementation("org.slf4j:slf4j-simple:2.0.16")

    // Testing
    testImplementation(libs.junit)
    testImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-test:1.9.0")
    testImplementation("org.json:json:20231013") // For JSON parsing in unit tests

    androidTestImplementation(libs.androidx.junit)
    androidTestImplementation(libs.androidx.espresso.core)
    androidTestImplementation(platform(libs.androidx.compose.bom))
    androidTestImplementation(libs.androidx.compose.ui.test.junit4)
    androidTestImplementation("io.mockk:mockk-android:1.13.13")

    debugImplementation(libs.androidx.compose.ui.tooling)
    debugImplementation(libs.androidx.compose.ui.test.manifest)
}
