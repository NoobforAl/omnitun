package dev.omnitun.app

import android.Manifest
import android.content.Intent
import android.content.pm.ApplicationInfo
import android.content.pm.PackageManager
import android.graphics.Bitmap
import android.graphics.Canvas
import android.net.Uri
import android.net.VpnService
import android.os.Build
import android.os.Bundle
import android.provider.Settings
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import java.io.ByteArrayOutputStream
import java.util.concurrent.Executors

/**
 * Bridges Flutter to the core over the "omnitun/core" channel:
 *  - start: launch the in-process core, returns {"addr","token"} for the control protocol
 *  - prepareVpn: show the system VPN consent dialog if needed, returns true when granted
 *  - prepareIkev2: register an IKEv2 profile with the system (its own consent dialog)
 *  - requestNotifications: ask for POST_NOTIFICATIONS (Android 13+)
 */
class MainActivity : FlutterActivity() {

    private val io = Executors.newSingleThreadExecutor()
    private var pendingVpn: MethodChannel.Result? = null

    /** A link or config file another app sent us, until Flutter has taken it (see "omnitun/incoming"). */
    private var incoming: String? = null
    private var incomingChannel: MethodChannel? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // Only a fresh start: after a rotation the same intent would import the file again.
        if (savedInstanceState == null) takeIncoming(intent)
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        takeIncoming(intent)
    }

    private fun takeIncoming(intent: Intent?) {
        val text = incomingText(intent) ?: return
        incoming = text
        incomingChannel?.invokeMethod("incoming", null) // the app is running: tell it now; else it asks at start
    }

    /** The text a VIEW or SEND intent carries: the link itself, or the contents of the file (up to 2 MB). */
    private fun incomingText(intent: Intent?): String? {
        intent ?: return null
        return when (intent.action) {
            Intent.ACTION_VIEW -> {
                val uri = intent.data ?: return null
                if (uri.scheme == "content" || uri.scheme == "file") readText(uri) else uri.toString()
            }
            Intent.ACTION_SEND -> {
                @Suppress("DEPRECATION")
                val stream: Uri? = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                    intent.getParcelableExtra(Intent.EXTRA_STREAM, Uri::class.java)
                } else {
                    intent.getParcelableExtra(Intent.EXTRA_STREAM)
                }
                stream?.let { readText(it) } ?: intent.getStringExtra(Intent.EXTRA_TEXT)
            }
            else -> null
        }
    }

    private fun readText(uri: Uri): String? = runCatching {
        contentResolver.openInputStream(uri)?.use { input ->
            val out = ByteArrayOutputStream()
            val buf = ByteArray(8192)
            while (true) {
                val n = input.read(buf)
                if (n < 0) break
                out.write(buf, 0, n)
                if (out.size() > MAX_INCOMING) return@runCatching null // not a config file
            }
            out.toString(Charsets.UTF_8.name())
        }
    }.getOrNull()

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        incomingChannel = MethodChannel(flutterEngine.dartExecutor.binaryMessenger, "omnitun/incoming").also { ch ->
            ch.setMethodCallHandler { call, result ->
                if (call.method == "take") {
                    result.success(incoming)
                    incoming = null
                } else {
                    result.notImplemented()
                }
            }
        }
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
                "prepareIkev2" -> {
                    // The system asks once to allow this app's IKEv2 VPN profile.
                    if (!Ikev2.supported) {
                        result.error("UNSUPPORTED", "IKEv2 needs Android 11 or newer", null)
                    } else {
                        val intent = runCatching { Ikev2.provision(this, call.arguments as String) }
                            .getOrElse { result.error("BAD_PROFILE", it.message, null); return@setMethodCallHandler }
                        if (intent == null) {
                            result.success(true)
                        } else {
                            pendingVpn?.success(false)
                            pendingVpn = result
                            @Suppress("DEPRECATION")
                            startActivityForResult(intent, REQUEST_VPN)
                        }
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
                "saveLastConnect" -> {
                    OmnitunVpnService.saveLastConnect(this, call.arguments as String)
                    result.success(null)
                }
                "openVpnSettings" -> {
                    // Where "Always-on VPN" and "Block connections without VPN" are switched on.
                    val opened = runCatching { startActivity(Intent(Settings.ACTION_VPN_SETTINGS)) }.isSuccess
                    result.success(opened)
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
        private const val MAX_INCOMING = 2 * 1024 * 1024
    }
}
