package dev.omnitun.app

import android.Manifest
import android.content.Intent
import android.content.pm.ApplicationInfo
import android.content.pm.PackageManager
import android.graphics.Bitmap
import android.graphics.Canvas
import android.net.VpnService
import android.os.Build
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import java.io.ByteArrayOutputStream
import java.util.concurrent.Executors

/**
 * Bridges Flutter to the core over the "omnitun/core" channel:
 *  - start: launch the in-process core, returns {"addr","token"} for the control protocol
 *  - prepareVpn: show the system VPN consent dialog if needed, returns true when granted
 *  - requestNotifications: ask for POST_NOTIFICATIONS (Android 13+)
 */
class MainActivity : FlutterActivity() {

    private val io = Executors.newSingleThreadExecutor()
    private var pendingVpn: MethodChannel.Result? = null

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, "omnitun/core").setMethodCallHandler { call, result ->
            when (call.method) {
                "start" -> io.execute {
                    val res = runCatching { CorePlatform.start(this) }
                    runOnUiThread {
                        res.fold({ result.success(it) }, { result.error("START_FAILED", it.message, null) })
                    }
                }
                "prepareVpn" -> {
                    val intent = VpnService.prepare(this)
                    if (intent == null) {
                        result.success(true)
                    } else {
                        pendingVpn?.success(false)
                        pendingVpn = result
                        @Suppress("DEPRECATION")
                        startActivityForResult(intent, REQUEST_VPN)
                    }
                }
                "listApps" -> io.execute {
                    // [{package, label, system}] for the split-tunneling picker.
                    val res = runCatching {
                        val pm = packageManager
                        pm.getInstalledApplications(0)
                            .filter { it.packageName != packageName }
                            .map {
                                mapOf(
                                    "package" to it.packageName,
                                    "label" to pm.getApplicationLabel(it).toString(),
                                    // Preinstalled apps with a launcher icon (Chrome, YouTube…) are
                                    // ordinary apps to the user; only hide background system packages.
                                    "system" to ((it.flags and ApplicationInfo.FLAG_SYSTEM) != 0 &&
                                        pm.getLaunchIntentForPackage(it.packageName) == null),
                                )
                            }
                    }
                    runOnUiThread {
                        res.fold({ result.success(it) }, { result.error("LIST_APPS_FAILED", it.message, null) })
                    }
                }
                "appIcon" -> io.execute {
                    val pkg = call.arguments as String
                    val png = runCatching { iconPng(pkg) }.getOrNull()
                    runOnUiThread { result.success(png) }
                }
                "setStopOnClose" -> {
                    OmnitunVpnService.setStopOnClose(this, call.arguments as? Boolean ?: true)
                    result.success(null)
                }
                "requestNotifications" -> {
                    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
                        checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED
                    ) {
                        requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), REQUEST_NOTIFICATIONS)
                    }
                    result.success(null)
                }
                else -> result.notImplemented()
            }
        }
    }

    @Deprecated("Deprecated in Java")
    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        if (requestCode == REQUEST_VPN) {
            pendingVpn?.success(resultCode == RESULT_OK)
            pendingVpn = null
            return
        }
        @Suppress("DEPRECATION")
        super.onActivityResult(requestCode, resultCode, data)
    }

    /** The app's launcher icon as a 96 px PNG. */
    private fun iconPng(pkg: String): ByteArray {
        val drawable = packageManager.getApplicationIcon(pkg)
        val size = 96
        val bitmap = Bitmap.createBitmap(size, size, Bitmap.Config.ARGB_8888)
        val canvas = Canvas(bitmap)
        drawable.setBounds(0, 0, size, size)
        drawable.draw(canvas)
        return ByteArrayOutputStream().use { out ->
            bitmap.compress(Bitmap.CompressFormat.PNG, 100, out)
            out.toByteArray()
        }
    }

    companion object {
        private const val REQUEST_VPN = 1001
        private const val REQUEST_NOTIFICATIONS = 1002
    }
}
