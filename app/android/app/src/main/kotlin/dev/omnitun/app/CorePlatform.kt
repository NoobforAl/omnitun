package dev.omnitun.app

import android.content.Context
import android.net.ConnectivityManager
import android.net.LinkProperties
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.os.Handler
import android.os.HandlerThread
import dev.omnitun.mobile.DefaultNetworkListener
import dev.omnitun.mobile.Mobile
import dev.omnitun.mobile.Platform
import org.json.JSONArray
import org.json.JSONObject
import java.net.NetworkInterface

/** Android side of the Go core's Platform interface (see core/mobile). */
class CorePlatform(context: Context) : Platform {

    private val app = context.applicationContext
    private val cm = app.getSystemService(ConnectivityManager::class.java)
    private val thread = HandlerThread("omnitun-net").apply { start() }
    private val handler = Handler(thread.looper)
    private var callback: ConnectivityManager.NetworkCallback? = null

    override fun openTun(tunJSON: String): Int =
        OmnitunVpnService.startAndAwait(app).openTun(tunJSON)

    override fun protect(fd: Int) {
        // Without a running VPN nothing needs protecting (e.g. latency tests).
        val service = OmnitunVpnService.instance ?: return
        if (!service.protect(fd)) throw IllegalStateException("protect($fd) failed")
    }

    override fun stateChanged(stateJSON: String) {
        val state = runCatching { JSONObject(stateJSON) }.getOrNull() ?: return
        val service = OmnitunVpnService.instance ?: return
        when (val status = state.optString("status")) {
            "disconnected", "error" -> service.stopVpn()
            else -> service.updateStatus(
                status,
                state.optString("profile"),
                state.optLong("up_bps"),
                state.optLong("down_bps"),
            )
        }
    }

    override fun interfaces(): String {
        val out = JSONArray()
        for (network in cm.allNetworks) {
            val lp = cm.getLinkProperties(network) ?: continue
            val caps = cm.getNetworkCapabilities(network) ?: continue
            if (caps.hasTransport(NetworkCapabilities.TRANSPORT_VPN)) continue
            val name = lp.interfaceName ?: continue
            val ni = runCatching { NetworkInterface.getByName(name) }.getOrNull() ?: continue
            out.put(
                JSONObject()
                    .put("name", name)
                    .put("index", ni.index)
                    .put("mtu", runCatching { ni.mtu }.getOrDefault(1500))
                    .put("addresses", JSONArray(lp.linkAddresses.map { it.toString() }))
                    .put("flags", flags(ni))
                    .put("type", type(caps))
                    .put("dns", JSONArray(lp.dnsServers.map { it.hostAddress }))
                    .put("metered", !caps.hasCapability(NetworkCapabilities.NET_CAPABILITY_NOT_METERED)),
            )
        }
        return out.toString()
    }

    override fun startMonitor(listener: DefaultNetworkListener) {
        closeMonitor()
        // Track the best non-VPN network: our own VPN must never be the "default".
        val request = NetworkRequest.Builder()
            .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
            .addCapability(NetworkCapabilities.NET_CAPABILITY_NOT_VPN)
            .build()
        val cb = object : ConnectivityManager.NetworkCallback() {
            private var current: Network? = null

            override fun onAvailable(network: Network) {
                current = network
                report(listener, network)
            }

            override fun onLinkPropertiesChanged(network: Network, lp: LinkProperties) {
                if (network == current) report(listener, network)
            }

            override fun onCapabilitiesChanged(network: Network, caps: NetworkCapabilities) {
                if (network == current) report(listener, network)
            }

            override fun onLost(network: Network) {
                if (network == current) {
                    current = null
                    listener.updateDefaultInterface("", -1, false, false)
                }
            }
        }
        cm.requestNetwork(request, cb, handler)
        callback = cb
        // Report the current network before returning: sing-box resets connections on the first
        // default-interface update, so it must arrive while the engine is starting, not after.
        currentUnderlyingNetwork()?.let { report(listener, it) }
    }

    /** The active non-VPN network with internet access, if any. */
    private fun currentUnderlyingNetwork(): Network? {
        fun usable(n: Network) = cm.getNetworkCapabilities(n)?.let {
            it.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) &&
                !it.hasTransport(NetworkCapabilities.TRANSPORT_VPN)
        } == true
        cm.activeNetwork?.takeIf(::usable)?.let { return it }
        return cm.allNetworks.firstOrNull(::usable)
    }

    override fun closeMonitor() {
        callback?.let { runCatching { cm.unregisterNetworkCallback(it) } }
        callback = null
    }

    private fun report(listener: DefaultNetworkListener, network: Network) {
        val lp = cm.getLinkProperties(network) ?: return
        val name = lp.interfaceName ?: return
        val index = runCatching { NetworkInterface.getByName(name)?.index }.getOrNull() ?: return
        val caps = cm.getNetworkCapabilities(network)
        val expensive = caps?.hasCapability(NetworkCapabilities.NET_CAPABILITY_NOT_METERED) == false
        listener.updateDefaultInterface(name, index, expensive, false)
    }

    private fun type(caps: NetworkCapabilities): String = when {
        caps.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) -> "wifi"
        caps.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) -> "cellular"
        caps.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET) -> "ethernet"
        else -> "other"
    }

    /** unix IFF_* bits, decoded by the Go side. */
    private fun flags(ni: NetworkInterface): Int {
        var f = 0
        runCatching {
            if (ni.isUp) f = f or 0x1 or 0x40
            if (ni.isLoopback) f = f or 0x8
            if (ni.isPointToPoint) f = f or 0x10
            if (ni.supportsMulticast()) f = f or 0x1000
            if (!ni.isPointToPoint && !ni.isLoopback && ni.interfaceAddresses.any { it.broadcast != null }) f = f or 0x2
        }
        return f
    }

    companion object {
        @Volatile
        private var info: String? = null

        /** Starts the core once per process and returns {"addr","token"} JSON. */
        @Synchronized
        fun start(context: Context): String {
            info?.let { return it }
            val dataDir = context.filesDir.resolve("core").apply { mkdirs() }
            return Mobile.start(dataDir.absolutePath, CorePlatform(context)).also { info = it }
        }
    }
}
