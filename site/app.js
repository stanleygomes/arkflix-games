// Detecta o sistema de quem visita e destaca o download dele. Os links são
// fixos (.../releases/latest/download/<arquivo sem versão>): o GitHub sempre
// entrega o da última release. A API só mostra o número da versão (se cair ou
// passar do limite, nada quebra).
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

const DOWNLOAD = `https://github.com/${REPO}/releases/latest/download/arkflix-games-`

function showVersion(release) {
  document.getElementById('hud-version').textContent = release.tag_name
  const line = document.getElementById('release-line')
  line.innerHTML = ''
  line.append('Última versão: ')
  const a = document.createElement('a')
  a.href = release.html_url
  a.textContent = `${release.tag_name} · novidades`
  line.append(a)
}

function highlight(os) {
  if (!os) return
  document.querySelector(`[data-os="${os}"]`)?.classList.add('mine')
  const cta = document.getElementById('cta-main')
  document.getElementById('cta-label').textContent = LABELS[os]
  if (os === 'linux') {
    cta.href = '#baixar' // no Linux o melhor é o comando, que traz os emuladores
    document.getElementById('cta-note').textContent = 'Um comando instala o app e os emuladores · também pra Windows e macOS'
    return
  }
  cta.href = DOWNLOAD + MAIN_ASSET[os]
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

highlight(detectOS())
latest().then(showVersion).catch(() => {})
