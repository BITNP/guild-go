import * as THREE from 'three'

export function createLabelTexture(title: string, completed: boolean, virtual: boolean): THREE.CanvasTexture {
  const canvas = document.createElement('canvas')
  canvas.width = 512
  canvas.height = 128
  const context = canvas.getContext('2d')
  if (!context) throw new Error('Canvas 2D context is unavailable')

  context.clearRect(0, 0, canvas.width, canvas.height)
  context.fillStyle = virtual ? '#245e91' : completed ? '#176847' : '#203d57'
  context.font = '600 32px "Noto Sans SC", "PingFang SC", sans-serif'
  context.textBaseline = 'middle'

  const lines = wrapText(context, title || '未命名节点', 450, 2)
  const lineHeight = 37
  const firstY = lines.length === 1 ? 64 : 45
  lines.forEach((line, index) => {
    context.fillText(line, 34, firstY + index * lineHeight)
  })

  context.fillStyle = virtual ? '#2f74a8' : completed ? '#2d8a62' : '#397eae'
  context.fillRect(0, 36, 5, 56)
  const texture = new THREE.CanvasTexture(canvas)
  texture.colorSpace = THREE.SRGBColorSpace
  texture.minFilter = THREE.LinearMipmapLinearFilter
  texture.magFilter = THREE.LinearFilter
  return texture
}

export function createEdgeGeometry(
  from: THREE.Vector3,
  to: THREE.Vector3,
): THREE.BufferGeometry {
  const middle = from.clone().lerp(to, 0.5)
  middle.y += 0.45 + from.distanceTo(to) * 0.018
  const curve = new THREE.QuadraticBezierCurve3(
    from.clone().setY(0.82),
    middle,
    to.clone().setY(0.82),
  )
  return new THREE.BufferGeometry().setFromPoints(curve.getPoints(28))
}

export function createDust(): THREE.Points<THREE.BufferGeometry, THREE.PointsMaterial> {
  const positions = new Float32Array(900 * 3)
  for (let index = 0; index < 900; index++) {
    const radius = 35 + Math.random() * 210
    const angle = Math.random() * Math.PI * 2
    positions[index * 3] = Math.cos(angle) * radius
    positions[index * 3 + 1] = Math.random() * 80 - 8
    positions[index * 3 + 2] = Math.sin(angle) * radius
  }
  const geometry = new THREE.BufferGeometry()
  geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3))
  const material = new THREE.PointsMaterial({
    color: 0x7894aa,
    size: 0.24,
    transparent: true,
    opacity: 0.2,
    depthWrite: false,
    blending: THREE.NormalBlending,
  })
  return new THREE.Points(geometry, material)
}

function wrapText(
  context: CanvasRenderingContext2D,
  text: string,
  maxWidth: number,
  maxLines: number,
): string[] {
  const characters = Array.from(text.trim())
  const lines: string[] = []
  let current = ''
  for (const character of characters) {
    const candidate = current + character
    if (current && context.measureText(candidate).width > maxWidth) {
      lines.push(current)
      current = character
      if (lines.length === maxLines) break
    } else {
      current = candidate
    }
  }
  if (lines.length < maxLines && current) lines.push(current)
  if (lines.length === maxLines && characters.join('') !== lines.join('')) {
    let last = lines[maxLines - 1] ?? ''
    while (last && context.measureText(`${last}…`).width > maxWidth) {
      last = Array.from(last).slice(0, -1).join('')
    }
    lines[maxLines - 1] = `${last}…`
  }
  return lines.length > 0 ? lines : ['未命名节点']
}
