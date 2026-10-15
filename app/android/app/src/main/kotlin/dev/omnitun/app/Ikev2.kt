package dev.omnitun.app

import android.content.Context
import android.content.Intent
import android.net.ConnectivityManager
import android.net.Ikev2VpnProfile
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.net.VpnManager
import android.os.Build
import android.os.Process
import android.util.Base64
import android.util.Log
import androidx.annotation.RequiresApi
import dev.omnitun.mobile.Mobile
import org.json.JSONObject
import java.security.KeyFactory
import java.security.cert.CertificateFactory
import java.security.cert.X509Certificate
import java.security.spec.PKCS8EncodedKeySpec

/**
 * IKEv2/IPsec through Android's own IKEv2 client (VpnManager, Android 11+).
 * The OS owns that tunnel; the Go engine starts/stops it through
 * CorePlatform and gets its state from [watch].
 */
object Ikev2 {
    private const val TAG = "OmnitunIkev2"

    @Volatile
    private var callback: ConnectivityManager.NetworkCallback? = null

    val supported: Boolean get() = Build.VERSION.SDK_INT >= Build.VERSION_CODES.R

    /** Builds the platform profile from the core's profile JSON. */
    @RequiresApi(Build.VERSION_CODES.R)
    fun build(json: String): Ikev2VpnProfile {
        val p = JSONObject(json)
        val o = p.optJSONObject("ikev2") ?: throw IllegalArgumentException("not an IKEv2 profile")
        val server = p.getString("server")
        val user = p.optString("username")
        val auth = o.optString("auth", "eap")
        val localId = o.optString("local_id").ifEmpty { if (auth == "eap" && user.isNotEmpty()) user else server }
        val ca = o.optString("ca_cert").takeIf { it.isNotEmpty() }?.let(::certificate)
        val b = Ikev2VpnProfile.Builder(server, localId)
        when (auth) {
            "psk" -> b.setAuthPsk(o.getString("psk").toByteArray())
            "eap" -> {
                val pass = p.optString("password")
                require(user.isNotEmpty() && pass.isNotEmpty()) { "IKEv2 needs a username and password" }
                b.setAuthUsernamePassword(user, pass, ca)
            }
            "cert" -> b.setAuthDigitalSignature(certificate(o.getString("client_cert")), privateKey(o.getString("client_key")), ca)
            else -> throw IllegalArgumentException("unknown IKEv2 auth $auth")
        }
        o.optInt("mtu").takeIf { it >= 1280 }?.let { b.setMaxMtu(it) }
        b.setBypassable(false)
        return b.build()
    }

    /** Registers the profile; returns the consent Intent when the user must approve it. */
    @RequiresApi(Build.VERSION_CODES.R)
    fun provision(context: Context, json: String): Intent? =
        context.getSystemService(VpnManager::class.java).provisionVpnProfile(build(json))

    @RequiresApi(Build.VERSION_CODES.R)
    fun start(context: Context, json: String) {
        val app = context.applicationContext
        // Our VpnService and the platform VPN can't both run; stop ours without
        // telling the core (its revoke handler would stop the engine).
        OmnitunVpnService.instance?.stopVpn()
        if (provision(app, json) != null) {
            throw IllegalStateException("Allow the IKEv2 VPN profile first (connect again from the app)")
        }
        watch(app)
        app.getSystemService(VpnManager::class.java).startProvisionedVpnProfile()
    }

    @RequiresApi(Build.VERSION_CODES.R)
    fun stop(context: Context) {
        unwatch(context)
        runCatching { context.applicationContext.getSystemService(VpnManager::class.java).stopProvisionedVpnProfile() }
    }

    /** Reports our app's VPN network appearing/disappearing to the core. */
    private fun watch(context: Context) {
        unwatch(context)
        val cm = context.getSystemService(ConnectivityManager::class.java)
        val request = NetworkRequest.Builder()
            .addTransportType(NetworkCapabilities.TRANSPORT_VPN)
            .removeCapability(NetworkCapabilities.NET_CAPABILITY_NOT_VPN)
            .build()
        val cb = object : ConnectivityManager.NetworkCallback() {
            private var up: Network? = null

            override fun onCapabilitiesChanged(network: Network, caps: NetworkCapabilities) {
                if (up == null && caps.ownerUid == Process.myUid()) {
                    up = network
                    Log.i(TAG, "IKEv2 tunnel up")
                    Thread { Mobile.platformVPNStatus("connected", "") }.start()
                }
            }

            override fun onLost(network: Network) {
                if (network == up) {
                    up = null
                    Log.i(TAG, "IKEv2 tunnel down")
                    Thread { Mobile.platformVPNStatus("disconnected", "the IKEv2 connection was lost") }.start()
                }
            }
        }
        cm.registerNetworkCallback(request, cb)
        callback = cb
    }

    private fun unwatch(context: Context) {
        callback?.let { cb ->
            runCatching { context.getSystemService(ConnectivityManager::class.java).unregisterNetworkCallback(cb) }
        }
        callback = null
    }

    private fun pemBody(pem: String): ByteArray = Base64.decode(
        pem.lines().filterNot { it.startsWith("-----") }.joinToString(""), Base64.DEFAULT,
    )

    private fun certificate(pem: String): X509Certificate =
        CertificateFactory.getInstance("X.509").generateCertificate(pem.byteInputStream()) as X509Certificate

    private fun privateKey(pem: String) = pemBody(pem).let { der ->
        val spec = PKCS8EncodedKeySpec(der)
        runCatching { KeyFactory.getInstance("RSA").generatePrivate(spec) }
            .getOrElse { KeyFactory.getInstance("EC").generatePrivate(spec) }
    }
}
