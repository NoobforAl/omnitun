# The Go core (gomobile) calls these classes and methods by name over JNI.
-keep class go.** { *; }
-keep class dev.omnitun.mobile.** { *; }
# Kotlin implementations of Go interfaces (Platform, DefaultNetworkListener).
-keep class dev.omnitun.app.** { *; }
