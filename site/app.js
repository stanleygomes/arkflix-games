// Detecta o sistema de quem visita e troca os links pelos arquivos da última
// release (API pública do GitHub). Sem a API, os links vão pra página de releases.
const REPO = 'stanleygomes/arkflix-games'
const LABELS = { windows: 'Baixar pra Windows', macos: 'Baixar pra macOS', linux: 'Baixar pra Linux' }
const MAIN_ASSET = { windows: 'windows-x86_64-setup.exe', macos: 'macos-universal.dmg', linux: 'linux-x86_64.tar.gz' }

function detectOS() {
  const ua = `${navigator.userAgentData?.platform ?? ''} ${navigator.platform ?? ''} ${navigator.userAgent}`.toLowerCase()
  if (/android|iphone|ipad/.test(ua)) return null // celular: mostra as três opções
  if (ua.includes('win')) return 'windows'
  if (ua.includes('mac')) return 'macos'
  if (ua.includes('linux') || ua.includes('x11')) return 'linux'
  return null
}

async function latest() {
  const res = await fetch(`https://api.github.com/repos/${REPO}/releases/latest`, { headers: { Accept: 'application/vnd.github+json' } })
  if (!res.ok) throw new Error(`GitHub ${res.status}`)
  return res.json()
}

function wire(release) {
  const assets = release.assets ?? []
  const find = (suffix) => assets.find((a) => a.name.endsWith(suffix))
  for (const link of document.querySelectorAll('[data-asset]')) {
    const asset = find(link.dataset.asset)
    if (asset) {
      link.href = asset.browser_download_url
      link.title = `${asset.name} (${(asset.size / 1048576).toFixed(1)} MB)`
    }
  }
  document.getElementById('hud-version').textContent = release.tag_name
  const line = document.getElementById('release-line')
  line.innerHTML = ''
  line.append('Última versão: ')
  const a = document.createElement('a')
  a.href = release.html_url
  a.textContent = `${release.tag_name} · novidades`
  line.append(a)
  return find
}

function highlight(os, find) {
  if (!os) return
  document.querySelector(`[data-os="${os}"]`)?.classList.add('mine')
  const cta = document.getElementById('cta-main')
  document.getElementById('cta-label').textContent = LABELS[os]
  if (os === 'linux') {
    cta.href = '#baixar' // no Linux o melhor é o comando, que traz os emuladores
    document.getElementById('cta-note').textContent = 'Um comando instala o app e os emuladores · também pra Windows e macOS'
    return
  }
  const asset = find?.(MAIN_ASSET[os])
  if (asset) cta.href = asset.browser_download_url
  document.getElementById('cta-note').textContent = 'Grátis e de código aberto · também pra outros sistemas, mais abaixo'
}

document.getElementById('copy-cmd')?.addEventListener('click', async (ev) => {
  const text = document.getElementById('linux-cmd').textContent
  try {
    await navigator.clipboard.writeText(text)
    ev.currentTarget.textContent = 'Copiado!'
  } catch {
    ev.currentTarget.textContent = 'Selecione e copie'
  }
})

const os = detectOS()
latest()
  .then((release) => highlight(os, wire(release)))
  .catch(() => highlight(os, null))
