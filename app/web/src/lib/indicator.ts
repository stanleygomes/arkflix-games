// Luz neon que desliza até a aba ativa ([aria-current="page"]) dentro do
// contêiner. Uso: <nav use:indicator={chave}> (a chave muda quando a aba muda).
export function indicator(node: HTMLElement, _key?: unknown) {
  const light = document.createElement('span')
  light.className = 'tab-light'
  light.setAttribute('aria-hidden', 'true')
  node.prepend(light)
  const place = () => {
    const tab = node.querySelector<HTMLElement>('[aria-current="page"]')
    if (!tab) {
      light.style.opacity = '0'
      return
    }
    light.style.opacity = '1'
    light.style.width = `${tab.offsetWidth}px`
    light.style.height = `${tab.offsetHeight}px`
    light.style.transform = `translate(${tab.offsetLeft}px, ${tab.offsetTop}px)`
  }
  const ro = new ResizeObserver(place)
  ro.observe(node)
  queueMicrotask(place)
  return {
    update: () => queueMicrotask(place),
    destroy: () => {
      ro.disconnect()
      light.remove()
    },
  }
}
