const { AndroidConfig, withAndroidManifest, withDangerousMod } = require('expo/config-plugins')
const fs = require('fs')
const path = require('path')

/**
 * Mengizinkan HTTP polos HANYA untuk host development yang disebut eksplisit.
 *
 * Android 9+ memblokir HTTP polos secara default, sehingga backend Go yang
 * jalan di laptop tidak dapat dihubungi dari HP tanpa ini. Build production
 * tidak memuat plugin ini sama sekali dan tetap HTTPS-only.
 *
 * @param {object} config
 * @param {{ hosts: string[] }} props host development, contoh ['192.168.1.162']
 */
module.exports = function withDevCleartext(config, props) {
  const hosts = props?.hosts ?? []
  if (hosts.length === 0) {
    throw new Error('withDevCleartext: props.hosts wajib berisi minimal satu host')
  }

  config = withDangerousMod(config, [
    'android',
    async (config) => {
      const xmlDir = path.join(config.modRequest.platformProjectRoot, 'app/src/main/res/xml')
      fs.mkdirSync(xmlDir, { recursive: true })

      const domains = hosts
        .map((h) => `        <domain includeSubdomains="true">${h}</domain>`)
        .join('\n')

      // base-config menolak cleartext untuk semua host lain, termasuk internet.
      const xml = `<?xml version="1.0" encoding="utf-8"?>
<network-security-config>
    <base-config cleartextTrafficPermitted="false" />
    <domain-config cleartextTrafficPermitted="true">
${domains}
    </domain-config>
</network-security-config>
`
      fs.writeFileSync(path.join(xmlDir, 'network_security_config.xml'), xml)
      return config
    },
  ])

  return withAndroidManifest(config, (config) => {
    const app = AndroidConfig.Manifest.getMainApplicationOrThrow(config.modResults)
    app.$['android:networkSecurityConfig'] = '@xml/network_security_config'
    return config
  })
}
