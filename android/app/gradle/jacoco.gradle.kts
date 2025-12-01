// JaCoCo configuration
// Note: The jacoco {} extension must remain in build.gradle.kts as it requires the plugin context.
// This file only contains the custom report task.

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
