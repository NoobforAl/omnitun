package dev.omnitun.app

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.ServiceInfo
import android.net.ProxyInfo
import android.net.VpnService
import android.os.Build
import android.os.ParcelFileDescriptor
import android.util.Log
import dev.omnitun.mobile.Mobile
import org.json.JSONArray
import org.json.JSONObject
import java.net.InetAddress
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit

/**
 * Holds the Android VPN interface for the Go core. The core asks for a TUN via
 * [CorePlatform.openTun]; this service starts in the foreground, builds the
 * interface and hands the fd back. It stops itself when the core reports the
 * tunnel stopped, when the user taps Disconnect, or when the OS revokes the VPN.
 */
class OmnitunVpnService : VpnService() {

    private var tun: ParcelFileDescriptor? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_DISCONNECT -> {
                Thread { Mobile.stopTunnel() }.start()
                return START_NOT_STICKY
            }
            else -> {
                startForegroundCompat(buildNotification(getString(R.string.vpn_connecting)))
                instance = this
                ready.countDown()
            }
        }
        // Never restart on our own: a VPN must only run while the core asks for it.
        return START_NOT_STICKY
    }

    /** Builds the VPN interface from the core's tun config JSON and returns its fd. */
    fun openTun(json: String): Int {
        val cfg = JSONObject(json)
        val builder = Builder().setSession(getString(R.string.app_name))
        val mtu = cfg.optInt("mtu", 0)
        if (mtu > 0) builder.setMtu(mtu)

        cfg.strings("inet4_address").forEach { builder.addAddress(it.addr(), it.bits()) }
        cfg.strings("inet6_address").forEach { builder.addAddress(it.addr(), it.bits()) }

        if (cfg.optBoolean("auto_route", true)) {
            val v4 = cfg.strings("inet4_route_range").ifEmpty { listOf("0.0.0.0/0") }
            val v6 = cfg.strings("inet6_route_range").ifEmpty { listOf("::/0") }
            (v4 + v6).forEach { builder.addRoute(it.addr(), it.bits()) }
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                (cfg.strings("inet4_route_exclude") + cfg.strings("inet6_route_exclude")).forEach {
                    builder.excludeRoute(android.net.IpPrefix(InetAddress.getByName(it.addr()), it.bits()))
                }
            }
        }
        cfg.strings("dns_servers").forEach { builder.addDnsServer(it) }

        val include = cfg.strings("include_package")
        val exclude = cfg.strings("exclude_package")
        if (include.isNotEmpty()) {
            include.forEach { runCatching { builder.addAllowedApplication(it) } }
        } else {
            exclude.forEach { runCatching { builder.addDisallowedApplication(it) } }
        }

        cfg.optJSONObject("http_proxy")?.let { proxy ->
            builder.setHttpProxy(
                ProxyInfo.buildDirectProxy(
                    proxy.getString("server"),
                    proxy.getInt("port"),
                    proxy.optJSONArray("bypass_domain").toList(),
                ),
            )
        }
        builder.setMetered(false)

        val pfd = builder.establish() ?: throw IllegalStateException("VPN permission not granted")
        tun?.close()
        tun = pfd
        updateNotification(getString(R.string.vpn_connected))
        return pfd.fd
    }

    /** Called when the core reports the tunnel stopped. */
    fun stopVpn() {
        tun?.close()
        tun = null
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    /** The user swiped the app away from Recents. */
    override fun onTaskRemoved(rootIntent: Intent?) {
        if (stopOnClose(this)) {
            Log.i(TAG, "app task removed, stopping VPN")
            Thread { Mobile.stopTunnel() }.start()
            stopVpn()
        }
        super.onTaskRemoved(rootIntent)
    }

    override fun onRevoke() {
        // Another VPN took over or the user revoked us in system settings.
        Log.i(TAG, "VPN revoked by the system")
        Thread { Mobile.stopTunnel() }.start()
        stopVpn()
    }

    override fun onDestroy() {
        tun?.close()
        tun = null
        if (instance === this) {
            instance = null
            ready = CountDownLatch(1)
        }
        super.onDestroy()
    }

    private fun startForegroundCompat(n: Notification) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE) {
            startForeground(NOTIFICATION_ID, n, ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE)
        } else {
            startForeground(NOTIFICATION_ID, n)
        }
    }

    private fun updateNotification(text: String) {
        getSystemService(NotificationManager::class.java).notify(NOTIFICATION_ID, buildNotification(text))
    }

    private fun buildNotification(text: String): Notification {
        val nm = getSystemService(NotificationManager::class.java)
        if (nm.getNotificationChannel(CHANNEL) == null) {
            nm.createNotificationChannel(
                NotificationChannel(CHANNEL, getString(R.string.vpn_channel), NotificationManager.IMPORTANCE_LOW),
            )
        }
        val flags = PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
        val open = PendingIntent.getActivity(this, 0, Intent(this, MainActivity::class.java), flags)
        val disconnect = PendingIntent.getService(
            this, 1, Intent(this, OmnitunVpnService::class.java).setAction(ACTION_DISCONNECT), flags,
        )
        return Notification.Builder(this, CHANNEL)
            .setSmallIcon(R.drawable.ic_vpn_key)
            .setContentTitle(getString(R.string.app_name))
            .setContentText(text)
            .setContentIntent(open)
            .setOngoing(true)
            .addAction(Notification.Action.Builder(null, getString(R.string.vpn_disconnect), disconnect).build())
            .build()
    }

    companion object {
        private const val TAG = "OmnitunVpn"
        private const val CHANNEL = "vpn"
        private const val NOTIFICATION_ID = 1
        private const val ACTION_DISCONNECT = "dev.omnitun.app.DISCONNECT"

        private const val PREFS = "omnitun"
        private const val KEY_STOP_ON_CLOSE = "stop_on_close"

        @Volatile
        var instance: OmnitunVpnService? = null
            private set

        /** "Stop VPN when app is closed", mirrored from the Flutter settings. */
        fun stopOnClose(context: Context): Boolean =
            context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getBoolean(KEY_STOP_ON_CLOSE, true)

        fun setStopOnClose(context: Context, value: Boolean) {
            context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit().putBoolean(KEY_STOP_ON_CLOSE, value).apply()
        }

        @Volatile
        private var ready = CountDownLatch(1)

        /** Starts the service in the foreground and waits until it is ready. */
        fun startAndAwait(context: Context, timeoutSeconds: Long = 10): OmnitunVpnService {
            instance?.let { return it }
            context.startForegroundService(Intent(context, OmnitunVpnService::class.java))
            if (!ready.await(timeoutSeconds, TimeUnit.SECONDS)) {
                throw IllegalStateException("VPN service did not start")
            }
            return instance ?: throw IllegalStateException("VPN service did not start")
        }
    }
}

private fun JSONObject.strings(key: String): List<String> = optJSONArray(key).toList()

private fun JSONArray?.toList(): List<String> =
    if (this == null) emptyList() else List(length()) { getString(it) }

private fun String.addr(): String = substringBefore('/')

private fun String.bits(): Int = substringAfter('/').toInt()
