import * as THREE from 'three'
import type { Comment } from '@/lib/api'
import type { GraphNeighborhood, GraphNode } from '@/graph/types'
import {
  layoutGraph,
  VIRTUAL_ROOT_ID,
  type GraphLayout,
} from './layout'
import { CommentLayer } from './CommentLayer'
import {
  createDust,
  createEdgeGeometry,
  createLabelTexture,
  createNodeGlowTexture,
  LABEL_ASPECT_RATIO,
} from './visuals'

interface SceneCallbacks {
  onSelect: (id: number) => void
  onBlank: () => void
  onCommentsNeedOlder: () => void
}

interface NodeVisual {
  group: THREE.Group
  plate: THREE.Mesh<THREE.CylinderGeometry, THREE.MeshStandardMaterial>
  cap: THREE.Mesh<THREE.CylinderGeometry, THREE.MeshStandardMaterial>
  crystal: THREE.Mesh<THREE.OctahedronGeometry, THREE.MeshStandardMaterial>
  innerRing: THREE.Mesh<THREE.RingGeometry, THREE.MeshBasicMaterial>
  outerRing: THREE.Mesh<THREE.RingGeometry, THREE.MeshBasicMaterial>
  glow: THREE.Sprite
  particles: THREE.Points<THREE.BufferGeometry, THREE.PointsMaterial>
  particleSeeds: Float32Array
  label: THREE.Sprite
  labelTargetPosition: THREE.Vector3
  radius: number
  baseEmissive: {
    plate: number
    cap: number
    crystal: number
  }
  targetPosition: THREE.Vector3
  targetOpacity: number
  opacity: number
  hoverAmount: number
  labelScale: number
  labelEmphasis: number
  labelOpacity: number
  targetLabelOpacity: number
}

interface EdgeVisual {
  line: THREE.Line<THREE.BufferGeometry, THREE.LineBasicMaterial>
  parentId: number
  childId: number
  targetOpacity: number
}

interface LabelCandidate {
  id: number
  view: NodeVisual
  priority: number
  left: number
  right: number
  top: number
  bottom: number
  readable: boolean
}

const LABEL_GAP = 10
const LABEL_REAR_OFFSET = 2.15
const LABEL_DEFAULT_OFFSET = 1.7
const INITIAL_PITCH = THREE.MathUtils.degToRad(60)
const INITIAL_YAW = THREE.MathUtils.degToRad(24)
const INTRO_DURATION_MS = 1700
const INTRO_START_DISTANCE_MULTIPLIER = 3.2
const LAYOUT_ROTATION_SPEED = THREE.MathUtils.degToRad(0.45)
const HOVER_HIT_RADIUS_PX = 24
const HOVER_FADE_SPEED = 9
const LABEL_ZOOM_HIDE_THRESHOLD = 0.68
const HOVER_PARTICLE_COUNT = 18
const HOVER_PARTICLE_STRIDE = 5

export class GraphScene {
  private readonly container: HTMLElement
  private readonly callbacks: SceneCallbacks
  private readonly scene = new THREE.Scene()
  private readonly camera = new THREE.PerspectiveCamera(44, 1, 0.1, 1800)
  private readonly renderer: THREE.WebGLRenderer
  private readonly world = new THREE.Group()
  private readonly nodeViews = new Map<number, NodeVisual>()
  private readonly edgeViews = new Map<string, EdgeVisual>()
  private readonly dataNodes = new Map<number, GraphNode>()
  private readonly extraNodes = new Map<number, GraphNode>()
  private readonly plateGeometry = new THREE.CylinderGeometry(1, 1, 0.16, 6)
  private readonly capGeometry = new THREE.CylinderGeometry(0.72, 0.72, 0.3, 6)
  private readonly crystalGeometry = new THREE.OctahedronGeometry(0.52, 0)
  private readonly innerRingGeometry = new THREE.RingGeometry(1.06, 1.15, 64)
  private readonly outerRingGeometry = new THREE.RingGeometry(
    1.36,
    1.41,
    64,
    1,
    0.22,
    Math.PI * 1.31,
  )
  private readonly glowTexture = createNodeGlowTexture()
  private readonly commentLayer = new CommentLayer()
  private readonly raycaster = new THREE.Raycaster()
  private readonly pointerNdc = new THREE.Vector2()
  private readonly pointerClient = new THREE.Vector2()
  private readonly pickCenter = new THREE.Vector3()
  private readonly pickOffset = new THREE.Vector3()
  private readonly viewTarget = new THREE.Vector3()
  private readonly desiredTarget = new THREE.Vector3()
  private readonly dragRight = new THREE.Vector3()
  private readonly dragForward = new THREE.Vector3()
  private readonly dust: THREE.Points
  private readonly resizeObserver: ResizeObserver
  private layout: GraphLayout = layoutGraph([])
  private focus: GraphNeighborhood | null = null
  private focusedIds = new Set<number>()
  private selectedId: number | null = null
  private hoveredId: number | null = null
  private commentNodeId: number | null = null
  private snapshot: {
    target: THREE.Vector3
    distance: number
    pitch: number
  } | null = null
  private distance = 90
  private desiredDistance = 90
  private fitDistance = 90
  private pitch = INITIAL_PITCH
  private desiredPitch = this.pitch
  private yaw = INITIAL_YAW
  private desiredYaw = this.yaw
  private introStarted = false
  private introActive = false
  private introStartedAt = 0
  private introFromDistance = 90
  private introToDistance = 90
  private introFromPitch = 0
  private introToPitch = INITIAL_PITCH
  private introFromYaw = INITIAL_YAW
  private introToYaw = INITIAL_YAW
  private focusNodePosition: THREE.Vector3 | null = null
  private pointerX = 0
  private pointerY = 0
  private pointerInside = false
  private smoothPointerX = 0
  private smoothPointerY = 0
  private dragging = false
  private dragDistance = 0
  private lastPointer = new THREE.Vector2()
  private hasInteracted = false
  private frameId = 0
  private lastFrame = performance.now()
  private disposed = false

  constructor(container: HTMLElement, callbacks: SceneCallbacks) {
    this.container = container
    this.callbacks = callbacks
    this.renderer = new THREE.WebGLRenderer({
      antialias: true,
      alpha: false,
      powerPreference: 'high-performance',
    })
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
    this.renderer.outputColorSpace = THREE.SRGBColorSpace
    this.renderer.toneMapping = THREE.ACESFilmicToneMapping
    this.renderer.toneMappingExposure = 1.1
    this.renderer.domElement.style.display = 'block'
    this.renderer.domElement.style.width = '100%'
    this.renderer.domElement.style.height = '100%'
    this.renderer.domElement.style.cursor = 'grab'
    this.renderer.domElement.style.touchAction = 'none'
    container.appendChild(this.renderer.domElement)

    this.scene.background = new THREE.Color(0x070a0d)
    this.scene.fog = new THREE.FogExp2(0x070a0d, 0.0024)
    this.world.add(this.createGround())
    this.scene.add(this.world)
    this.scene.add(this.commentLayer.group)
    this.dust = createDust()
    this.scene.add(this.dust)
    this.scene.add(new THREE.HemisphereLight(0xffe2a7, 0x071018, 2.2))
    const key = new THREE.DirectionalLight(0xffe0a0, 2.7)
    key.position.set(28, 60, 34)
    this.scene.add(key)
    const rim = new THREE.DirectionalLight(0x77b9d1, 1.2)
    rim.position.set(-36, 20, -28)
    this.scene.add(rim)

    this.camera.position.set(0, 88, 20)
    this.camera.lookAt(0, 0, 0)
    this.resizeObserver = new ResizeObserver(() => this.resize())
    this.resizeObserver.observe(container)
    this.bindEvents()
    this.resize()
    this.animate()
  }

  setNodes(nodes: GraphNode[]) {
    this.dataNodes.clear()
    for (const node of nodes) this.dataNodes.set(node.id, node)
    this.rebuildViews()
    if (!this.hasInteracted && !this.focus) this.fitView()
  }

  setComments(comments: Comment[], loading = false) {
    this.commentNodeId = comments[0]?.nodeId ?? null
    this.commentLayer.setComments(comments, loading)
    this.updateLabelTargets()
  }

  scrollComments(delta: number) {
    if (!this.focus) return
    if (this.commentLayer.scrollBy(delta)) {
      this.callbacks.onCommentsNeedOlder()
    }
  }

  setFocus(neighborhood: GraphNeighborhood | null) {
    if (!neighborhood) {
      this.clearFocus()
      return
    }
    this.introActive = false

    const related = new Map<number, GraphNode>()
    related.set(neighborhood.node.id, neighborhood.node)
    if (neighborhood.parent) related.set(neighborhood.parent.id, neighborhood.parent)
    for (const child of neighborhood.children) related.set(child.id, child)

    // Keep the entire ancestry in view so the camera turn lands on a visible
    // path from the selected node back to the guild core.
    let parentId = neighborhood.node.parentId
    const visited = new Set<number>([neighborhood.node.id])
    while (parentId != null && !visited.has(parentId)) {
      visited.add(parentId)
      const ancestor = this.dataNodes.get(parentId)
      if (!ancestor) break
      related.set(ancestor.id, ancestor)
      parentId = ancestor.parentId
    }

    for (const node of related.values()) this.extraNodes.set(node.id, node)
    const ids = new Set(related.keys())
    ids.add(VIRTUAL_ROOT_ID)

    if (!this.focus && !this.snapshot) {
      this.snapshot = {
        target: this.viewTarget.clone(),
        distance: this.distance,
        pitch: this.pitch,
      }
    }

    this.focus = neighborhood
    this.selectedId = neighborhood.node.id
    this.hoveredId = null
    this.focusedIds = ids
    this.world.updateMatrixWorld(true)
    this.focusNodePosition = new THREE.Vector3(
      this.layout.positions.get(this.selectedId)?.x ?? 0,
      0.9,
      this.layout.positions.get(this.selectedId)?.z ?? 0,
    ).applyMatrix4(this.world.matrixWorld)

    // The camera sits beyond the selected node on its radial axis. Looking back
    // toward the center therefore places the selected archive in front and the
    // guild origin deep in the scene.
    const radialYaw = this.focusNodePosition.lengthSq() > 0.01
      ? Math.atan2(this.focusNodePosition.x, this.focusNodePosition.z)
      : this.yaw
    this.desiredDistance = Math.max(16, Math.min(21, this.camera.fov / 2.2))
    this.desiredPitch = THREE.MathUtils.degToRad(21)
    this.desiredYaw = nearestAngle(this.yaw, radialYaw)
    this.desiredTarget.copy(this.focusTarget(
      this.focusNodePosition,
      this.desiredYaw,
      this.desiredPitch,
      this.desiredDistance,
    ))
    this.hasInteracted = true
    this.rebuildViews()
  }

  clearFocus() {
    if (!this.focus) return
    this.focus = null
    this.selectedId = null
    this.hoveredId = null
    this.focusNodePosition = null
    this.focusedIds.clear()
    this.hasInteracted = true
    if (this.snapshot) {
      this.desiredTarget.copy(this.snapshot.target)
      this.desiredDistance = this.snapshot.distance
      this.desiredPitch = this.snapshot.pitch
      // Keep the frozen layout bearing and return only the overview framing.
      this.desiredYaw = this.yaw
      this.snapshot = null
    }
    this.updateLabelTargets()
    this.applyVisualState()
  }

  fitView() {
    if (this.dataNodes.size === 0) return
    const verticalFov = THREE.MathUtils.degToRad(this.camera.fov)
    const horizontalFov = 2 * Math.atan(Math.tan(verticalFov / 2) * this.camera.aspect)
    const sceneRadius = Math.max(this.layout.radius + 6, 16)
    const horizontalDistance = sceneRadius / Math.tan(horizontalFov / 2)
    const verticalDistance = sceneRadius / Math.tan(verticalFov / 2)
    this.desiredTarget.set(0, 0.6, 0)
    this.viewTarget.copy(this.desiredTarget)
    this.desiredDistance = Math.max(28, horizontalDistance, verticalDistance) * 1.08
    this.fitDistance = this.desiredDistance
    this.desiredPitch = INITIAL_PITCH
    this.desiredYaw = nearestAngle(this.yaw, INITIAL_YAW)
    this.focusNodePosition = null
    if (!this.introStarted) {
      this.introStarted = true
      this.introActive = true
      this.introStartedAt = performance.now()
      this.distance = this.desiredDistance * INTRO_START_DISTANCE_MULTIPLIER
      this.pitch = 0
      this.yaw = this.desiredYaw
      this.introFromDistance = this.distance
      this.introToDistance = this.desiredDistance
      this.introFromPitch = 0
      this.introToPitch = this.desiredPitch
      this.introFromYaw = this.yaw
      this.introToYaw = this.desiredYaw
      return
    }
    if (this.introActive) {
      this.introToDistance = this.desiredDistance
      this.introToPitch = this.desiredPitch
      this.introToYaw = this.desiredYaw
      return
    }
    this.distance = this.desiredDistance
    this.pitch = this.desiredPitch
    this.yaw = this.desiredYaw
  }

  destroy() {
    this.disposed = true
    cancelAnimationFrame(this.frameId)
    this.resizeObserver.disconnect()
    this.unbindEvents()
    for (const id of [...this.nodeViews.keys()]) this.removeNode(id)
    for (const edge of this.edgeViews.values()) this.disposeEdge(edge)
    this.edgeViews.clear()
    this.commentLayer.dispose()
    this.world.traverse((object) => {
      if (object === this.world) return
      if (object instanceof THREE.Mesh || object instanceof THREE.Line) {
        object.geometry.dispose()
        const material: THREE.Material | THREE.Material[] = object.material
        const materials = Array.isArray(material) ? material : [material]
        materials.forEach((material) => material.dispose())
      }
    })
    this.plateGeometry.dispose()
    this.capGeometry.dispose()
    this.crystalGeometry.dispose()
    this.innerRingGeometry.dispose()
    this.outerRingGeometry.dispose()
    this.glowTexture.dispose()
    this.dust.geometry.dispose()
    if (Array.isArray(this.dust.material)) {
      this.dust.material.forEach((material) => material.dispose())
    } else {
      this.dust.material.dispose()
    }
    this.renderer.dispose()
    this.renderer.domElement.remove()
  }

  private rebuildViews() {
    const nodes = [...this.extraNodes.values()]
    const known = new Set(nodes.map((node) => node.id))
    for (const node of this.dataNodes.values()) {
      if (!known.has(node.id)) nodes.push(node)
    }
    nodes.push({
      id: VIRTUAL_ROOT_ID,
      parentId: null,
      title: '公会',
      sortOrder: -1,
      completedAt: null,
      childCount: nodes.filter((node) => node.parentId == null).length,
      commentCount: 0,
    })
    this.layout = layoutGraph(nodes)

    const activeIds = new Set(nodes.map((node) => node.id))
    for (const id of [...this.nodeViews.keys()]) {
      if (!activeIds.has(id)) this.removeNode(id)
    }
    for (const node of nodes) {
      const position = this.layout.positions.get(node.id)
      if (!position) continue
      let view = this.nodeViews.get(node.id)
      if (!view) {
        view = this.createNode(node)
        this.nodeViews.set(node.id, view)
        this.world.add(view.group)
      }
      view.targetPosition.set(position.x, 0.9, position.z)
    }
    this.updateLabelTargets()
    this.rebuildEdges()
    this.applyVisualState()
  }

  private createNode(node: GraphNode): NodeVisual {
    const virtual = node.id === VIRTUAL_ROOT_ID
    const radius = virtual ? 1.35 : 0.72
    const accent = virtual ? 0xf3ba46 : node.completedAt ? 0x58d7ad : 0xe0a83d
    const basePlateEmissive = virtual ? 0.32 : 0.12
    const baseCapEmissive = virtual ? 0.55 : 0.32
    const baseCrystalEmissive = virtual ? 1.9 : 1.1
    const plateMaterial = new THREE.MeshStandardMaterial({
      color: virtual ? 0x3b2b12 : 0x102027,
      emissive: accent,
      emissiveIntensity: basePlateEmissive,
      roughness: 0.32,
      metalness: 0.72,
      transparent: true,
      opacity: 0,
    })
    const capMaterial = new THREE.MeshStandardMaterial({
      color: accent,
      emissive: accent,
      emissiveIntensity: baseCapEmissive,
      roughness: 0.24,
      metalness: 0.66,
      transparent: true,
      opacity: 0,
    })
    const crystalMaterial = new THREE.MeshStandardMaterial({
      color: virtual ? 0xffe5a6 : 0xe8f4ec,
      emissive: accent,
      emissiveIntensity: baseCrystalEmissive,
      roughness: 0.1,
      metalness: 0.32,
      transparent: true,
      opacity: 0,
    })
    const lineMaterial = new THREE.MeshBasicMaterial({
      color: accent,
      transparent: true,
      opacity: 0,
      depthWrite: false,
      blending: THREE.AdditiveBlending,
      side: THREE.DoubleSide,
    })

    const group = new THREE.Group()
    const plate = new THREE.Mesh(this.plateGeometry, plateMaterial)
    plate.scale.set(radius, virtual ? 1.5 : 1, radius)
    plate.rotation.y = Math.PI / 6
    plate.userData.nodeId = node.id

    const cap = new THREE.Mesh(this.capGeometry, capMaterial)
    cap.scale.set(radius, virtual ? 1.45 : 1, radius)
    cap.position.y = 0.2
    cap.rotation.y = Math.PI / 6

    const crystal = new THREE.Mesh(this.crystalGeometry, crystalMaterial)
    crystal.scale.setScalar(radius * (virtual ? 1.18 : 0.9))
    crystal.position.y = virtual ? 0.72 : 0.48
    crystal.rotation.y = Math.PI / 4

    const innerRing = new THREE.Mesh(this.innerRingGeometry, lineMaterial.clone())
    innerRing.scale.setScalar(radius)
    innerRing.rotation.x = -Math.PI / 2
    innerRing.position.y = 0.17

    const outerRing = new THREE.Mesh(this.outerRingGeometry, lineMaterial.clone())
    outerRing.scale.setScalar(radius)
    outerRing.rotation.x = -Math.PI / 2
    outerRing.rotation.z = virtual ? 0.4 : Math.PI / 6
    outerRing.position.y = 0.205

    const glow = new THREE.Sprite(new THREE.SpriteMaterial({
      map: this.glowTexture,
      color: accent,
      transparent: true,
      opacity: 0,
      depthWrite: false,
      blending: THREE.AdditiveBlending,
    }))
    glow.scale.setScalar(radius * (virtual ? 4.8 : 4.1))
    glow.position.y = 0.28

    const particlePositions = new Float32Array(HOVER_PARTICLE_COUNT * 3)
    const particleSeeds = new Float32Array(HOVER_PARTICLE_COUNT * HOVER_PARTICLE_STRIDE)
    for (let index = 0; index < HOVER_PARTICLE_COUNT; index++) {
      const offset = index * HOVER_PARTICLE_STRIDE
      particleSeeds[offset] = Math.random() * Math.PI * 2
      particleSeeds[offset + 1] = 0.8 + Math.random() * 1.35
      particleSeeds[offset + 2] = 0.7 + Math.random() * 0.85
      particleSeeds[offset + 3] = Math.random()
      particleSeeds[offset + 4] = 1.35 + Math.random() * 1.25
    }
    const particleGeometry = new THREE.BufferGeometry()
    particleGeometry.setAttribute(
      'position',
      new THREE.BufferAttribute(particlePositions, 3),
    )
    const particles = new THREE.Points(
      particleGeometry,
      new THREE.PointsMaterial({
        map: this.glowTexture,
        color: accent,
        size: radius * 0.34,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        blending: THREE.AdditiveBlending,
        sizeAttenuation: true,
      }),
    )
    particles.frustumCulled = false
    particles.visible = false

    const labelMaterial = new THREE.SpriteMaterial({
      map: createLabelTexture(node.title, node.completedAt != null, virtual),
      transparent: true,
      opacity: 0,
      depthWrite: false,
      toneMapped: false,
    })
    const label = new THREE.Sprite(labelMaterial)
    label.center.set(0.5, 0)
    label.scale.set(4.8, 0.9, 1)
    const labelTargetPosition = new THREE.Vector3()

    group.add(plate, cap, crystal, innerRing, outerRing, glow, particles, label)
    const position = this.layout.positions.get(node.id)
    group.position.set(position?.x ?? 0, 0.9, position?.z ?? 0)
    return {
      group,
      plate,
      cap,
      crystal,
      innerRing,
      outerRing,
      glow,
      particles,
      particleSeeds,
      label,
      labelTargetPosition,
      radius,
      baseEmissive: {
        plate: basePlateEmissive,
        cap: baseCapEmissive,
        crystal: baseCrystalEmissive,
      },
      targetPosition: group.position.clone(),
      targetOpacity: 0,
      opacity: 0,
      hoverAmount: 0,
      labelScale: 1,
      labelEmphasis: 1,
      labelOpacity: 0,
      targetLabelOpacity: 0,
    }
  }

  private updateLabelTargets() {
    for (const [id, view] of this.nodeViews) {
      const position = this.layout.positions.get(id)
      if (!position) continue
      const virtual = id === VIRTUAL_ROOT_ID
      const moveBehindComments = !virtual &&
        id === this.selectedId &&
        this.commentNodeId === id
      const labelOffset = virtual
        ? 0
        : moveBehindComments
          ? -LABEL_REAR_OFFSET
          : LABEL_DEFAULT_OFFSET
      view.labelTargetPosition.set(
        Math.cos(position.angle) * labelOffset,
        virtual ? 1.85 : 1.42,
        Math.sin(position.angle) * labelOffset,
      )
      if (view.label.position.lengthSq() === 0) {
        view.label.position.copy(view.labelTargetPosition)
      }
    }
  }

  private removeNode(id: number) {
    const view = this.nodeViews.get(id)
    if (!view) return
    this.world.remove(view.group)
    view.label.material.map?.dispose()
    view.label.material.dispose()
    view.plate.material.dispose()
    view.cap.material.dispose()
    view.crystal.material.dispose()
    view.innerRing.material.dispose()
    view.outerRing.material.dispose()
    view.glow.material.dispose()
    view.particles.geometry.dispose()
    view.particles.material.dispose()
    this.nodeViews.delete(id)
  }

  private rebuildEdges() {
    for (const [id, edge] of this.edgeViews) {
      this.world.remove(edge.line)
      this.disposeEdge(edge)
      this.edgeViews.delete(id)
    }
    for (const edge of this.layout.edges) {
      const parent = this.layout.positions.get(edge.parentId)
      const child = this.layout.positions.get(edge.childId)
      if (!parent || !child) continue
      const material = new THREE.LineBasicMaterial({
        color: 0xd4a34f,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        blending: THREE.AdditiveBlending,
      })
      const geometry = createEdgeGeometry(
        new THREE.Vector3(parent.x, 0, parent.z),
        new THREE.Vector3(child.x, 0, child.z),
      )
      const line = new THREE.Line(geometry, material)
      this.world.add(line)
      this.edgeViews.set(edge.id, {
        line,
        parentId: edge.parentId,
        childId: edge.childId,
        targetOpacity: 0.34,
      })
    }
  }

  private disposeEdge(edge: EdgeVisual) {
    edge.line.geometry.dispose()
    edge.line.material.dispose()
  }

  private applyVisualState() {
    for (const [id, view] of this.nodeViews) {
      const visible = !this.focus || this.focusedIds.has(id)
      view.targetOpacity = visible
        ? id === this.selectedId
          ? 1
          : id === VIRTUAL_ROOT_ID
            ? 0.78
              : id === this.hoveredId
                ? 1
                : 0.92
        : 0
    }
    for (const edge of this.edgeViews.values()) {
      edge.targetOpacity = !this.focus
        ? 0.34
        : this.focusedIds.has(edge.parentId) && this.focusedIds.has(edge.childId)
          ? 0.86
          : 0
    }
  }

  private createGround(): THREE.Group {
    const ground = new THREE.Group()
    const plane = new THREE.Mesh(
      new THREE.PlaneGeometry(1000, 1000),
      new THREE.MeshBasicMaterial({ color: 0x070b0e }),
    )
    plane.rotation.x = -Math.PI / 2
    plane.position.y = -0.13
    const grid = new THREE.Mesh(
      new THREE.PlaneGeometry(1000, 1000),
      new THREE.ShaderMaterial({
        transparent: true,
        depthWrite: false,
        side: THREE.DoubleSide,
        uniforms: {
          minorColor: { value: new THREE.Color(0x55808c) },
          accentColor: { value: new THREE.Color(0xe5ad46) },
          ringSpacing: { value: 3.6 },
          fadeStart: { value: 44 },
          fadeEnd: { value: 160 },
        },
        vertexShader: `
          varying vec2 vGridPosition;
          varying vec3 vWorldPosition;
          void main() {
            vGridPosition = position.xy;
            vec4 worldPosition = modelMatrix * vec4(position, 1.0);
            vWorldPosition = worldPosition.xyz;
            gl_Position = projectionMatrix * viewMatrix * worldPosition;
          }
        `,
        fragmentShader: `
          uniform vec3 minorColor;
          uniform vec3 accentColor;
          uniform float ringSpacing;
          uniform float fadeStart;
          uniform float fadeEnd;
          varying vec2 vGridPosition;
          varying vec3 vWorldPosition;

          float gridLine(float coordinate, float spacing) {
            float value = coordinate / spacing;
            float derivative = max(fwidth(value), 0.0001);
            float distanceToLine = abs(fract(value - 0.5) - 0.5) / derivative;
            return 1.0 - min(distanceToLine, 1.0);
          }

          void main() {
            // Sample in graph-local coordinates so the polar frame is part of
            // the same rotating rig as the node layout.
            float radius = length(vGridPosition);
            float angle = atan(vGridPosition.y, vGridPosition.x);
            float minorRing = gridLine(radius, ringSpacing);
            float majorRing = gridLine(radius, ringSpacing * 5.0);
            float spokes = gridLine(angle * 12.0 / 3.14159265, 1.0);
            float distanceFromCamera = distance(vWorldPosition.xz, cameraPosition.xz);
            float distanceFade = 1.0 - smoothstep(fadeStart, fadeEnd, distanceFromCamera);
            float radialFade = smoothstep(2.4, 9.0, radius);
            float activeSector = smoothstep(
              -0.7,
              0.82,
              dot(normalize(vGridPosition + vec2(0.0001)), normalize(vec2(0.78, 0.62)))
            );
            float referenceDelta = atan(
              sin(angle - 1.5707963),
              cos(angle - 1.5707963)
            );
            float referenceRay = 1.0 - smoothstep(0.006, 0.026, abs(referenceDelta));
            float referenceFade = smoothstep(3.5, 8.5, radius);
            float minor = minorRing * (0.24 + activeSector * 0.18);
            float spokeLine = spokes * (0.16 + activeSector * 0.58);
            float accent = max(
              majorRing * (0.42 + activeSector * 0.58),
              spokeLine * 0.5
            );
            accent = max(
              accent,
              referenceRay * referenceFade * (0.34 + majorRing * 0.46)
            );
            float alpha = max(minor, accent) * distanceFade * radialFade;
            if (alpha < 0.002) discard;
            vec3 color = mix(minorColor, accentColor, accent);
            gl_FragColor = vec4(color, alpha * 0.72);
          }
        `,
      }),
    )
    grid.rotation.x = -Math.PI / 2
    grid.position.y = -0.1
    ground.add(plane, grid)
    return ground
  }

  private bindEvents() {
    this.renderer.domElement.addEventListener('pointerdown', this.onPointerDown)
    this.renderer.domElement.addEventListener('pointermove', this.onPointerMove)
    this.renderer.domElement.addEventListener('pointerup', this.onPointerUp)
    this.renderer.domElement.addEventListener('pointercancel', this.onPointerUp)
    this.renderer.domElement.addEventListener('pointerleave', this.onPointerLeave)
    this.renderer.domElement.addEventListener('wheel', this.onWheel, { passive: false })
  }

  private unbindEvents() {
    this.renderer.domElement.removeEventListener('pointerdown', this.onPointerDown)
    this.renderer.domElement.removeEventListener('pointermove', this.onPointerMove)
    this.renderer.domElement.removeEventListener('pointerup', this.onPointerUp)
    this.renderer.domElement.removeEventListener('pointercancel', this.onPointerUp)
    this.renderer.domElement.removeEventListener('pointerleave', this.onPointerLeave)
    this.renderer.domElement.removeEventListener('wheel', this.onWheel)
  }

  private onPointerDown = (event: PointerEvent) => {
    if (event.button !== 0) return
    this.introActive = false
    this.dragging = true
    this.pointerInside = true
    this.pointerClient.set(event.clientX, event.clientY)
    this.hoveredId = null
    this.applyVisualState()
    this.dragDistance = 0
    this.lastPointer.set(event.clientX, event.clientY)
    this.renderer.domElement.setPointerCapture(event.pointerId)
    this.renderer.domElement.style.cursor = 'grabbing'
  }

  private onPointerMove = (event: PointerEvent) => {
    const rect = this.renderer.domElement.getBoundingClientRect()
    this.pointerX = ((event.clientX - rect.left) / rect.width) * 2 - 1
    this.pointerY = ((event.clientY - rect.top) / rect.height) * 2 - 1
    this.pointerInside = true
    this.pointerClient.set(event.clientX, event.clientY)
    if (!this.dragging) {
      this.updateHoverFromPointer()
      this.renderer.domElement.style.cursor =
        this.hoveredId == null ? 'grab' : 'pointer'
      return
    }
    const dx = event.clientX - this.lastPointer.x
    const dy = event.clientY - this.lastPointer.y
    this.dragDistance += Math.abs(dx) + Math.abs(dy)
    this.lastPointer.set(event.clientX, event.clientY)
    if (this.focus || this.dragDistance < 3) return
    this.hasInteracted = true
    const scale = this.distance / Math.max(rect.height, 1) * 1.15
    this.dragRight.set(1, 0, 0).applyQuaternion(this.camera.quaternion)
    this.dragRight.y = 0
    if (this.dragRight.lengthSq() < 0.0001) {
      this.dragRight.set(Math.cos(this.yaw), 0, -Math.sin(this.yaw))
    } else {
      this.dragRight.normalize()
    }
    this.dragForward.set(0, 0, -1).applyQuaternion(this.camera.quaternion)
    this.dragForward.y = 0
    if (this.dragForward.lengthSq() < 0.0001) {
      this.dragForward.set(-Math.sin(this.yaw), 0, -Math.cos(this.yaw))
    } else {
      this.dragForward.normalize()
    }
    this.desiredTarget
      .addScaledVector(this.dragRight, -dx * scale)
      .addScaledVector(this.dragForward, dy * scale)
    this.renderer.domElement.style.cursor = 'grabbing'
  }

  private onPointerUp = (event: PointerEvent) => {
    if (!this.dragging) return
    this.dragging = false
    if (this.renderer.domElement.hasPointerCapture(event.pointerId)) {
      this.renderer.domElement.releasePointerCapture(event.pointerId)
    }
    if (this.dragDistance > 5) {
      this.renderer.domElement.style.cursor = 'grab'
      return
    }
    const id = this.pickNodeAt(event.clientX, event.clientY)
    this.renderer.domElement.style.cursor = id == null ? 'grab' : 'pointer'
    if (id == null) this.callbacks.onBlank()
    else this.callbacks.onSelect(id)
  }

  private onPointerLeave = () => {
    if (this.dragging) return
    this.pointerX = 0
    this.pointerY = 0
    this.pointerInside = false
    this.hoveredId = null
    this.applyVisualState()
  }

  private onWheel = (event: WheelEvent) => {
    if (this.focus) return
    this.introActive = false
    event.preventDefault()
    this.hasInteracted = true
    this.desiredDistance = THREE.MathUtils.clamp(
      this.desiredDistance * Math.exp(event.deltaY * 0.0011),
      16,
      440,
    )
  }

  private updateHoverFromPointer() {
    if (!this.pointerInside || this.dragging) return
    const hovered = this.pickNodeAt(
      this.pointerClient.x,
      this.pointerClient.y,
      true,
    )
    if (hovered === this.hoveredId) return
    this.hoveredId = hovered
    this.applyVisualState()
    this.renderer.domElement.style.cursor =
      hovered == null ? 'grab' : 'pointer'
  }

  private pickNodeAt(
    clientX: number,
    clientY: number,
    includeVirtual = false,
  ): number | null {
    const rect = this.renderer.domElement.getBoundingClientRect()
    this.pointerNdc.set(
      ((clientX - rect.left) / rect.width) * 2 - 1,
      -((clientY - rect.top) / rect.height) * 2 + 1,
    )
    this.raycaster.setFromCamera(this.pointerNdc, this.camera)
    const ray = this.raycaster.ray
    const viewportHeight = Math.max(1, rect.height)
    let nearestId: number | null = null
    let nearestRayDistance = Number.POSITIVE_INFINITY

    for (const [id, view] of this.nodeViews) {
      if (id === VIRTUAL_ROOT_ID && !includeVirtual) continue
      if (
        view.targetOpacity <= 0.01 ||
        view.opacity <= 0.08 ||
        !view.group.visible
      ) {
        continue
      }

      view.group.getWorldPosition(this.pickCenter)
      this.pickOffset.copy(this.pickCenter).sub(ray.origin)
      if (this.pickOffset.dot(ray.direction) <= 0) continue

      const rayDistance = ray.distanceToPoint(this.pickCenter)
      const cameraDistance = this.camera.position.distanceTo(this.pickCenter)
      const worldUnitsPerPixel = 2 * cameraDistance *
        Math.tan(THREE.MathUtils.degToRad(this.camera.fov / 2)) /
        viewportHeight
      // The fixed screen-space allowance expands in world units as the camera
      // moves away, keeping distant nodes practical to target.
      const hitDistance = view.radius * 1.35 +
        HOVER_HIT_RADIUS_PX * worldUnitsPerPixel
      if (rayDistance <= hitDistance && rayDistance < nearestRayDistance) {
        nearestId = id
        nearestRayDistance = rayDistance
      }
    }
    return nearestId
  }

  private focusTarget(
    nodePosition: THREE.Vector3,
    yaw: number,
    pitch: number,
    distance: number,
  ): THREE.Vector3 {
    const verticalExtent = 2 * distance * Math.tan(THREE.MathUtils.degToRad(this.camera.fov / 2))
    const horizontalExtent = verticalExtent * this.camera.aspect
    const offsetDirection = new THREE.Vector3(
      Math.sin(yaw) * Math.cos(pitch),
      Math.sin(pitch),
      Math.cos(yaw) * Math.cos(pitch),
    )
    const forward = offsetDirection.multiplyScalar(-1).normalize()
    const right = new THREE.Vector3().crossVectors(forward, new THREE.Vector3(0, 1, 0)).normalize()
    const screenUp = new THREE.Vector3().crossVectors(right, forward).normalize()
    return nodePosition.clone()
      .addScaledVector(right, horizontalExtent * 0.16)
      .addScaledVector(screenUp, verticalExtent * 0.18)
  }

  private resize() {
    const width = Math.max(1, this.container.clientWidth)
    const height = Math.max(1, this.container.clientHeight)
    this.camera.aspect = width / height
    this.camera.updateProjectionMatrix()
    this.renderer.setSize(width, height, false)
    if (this.focus && this.focusNodePosition) {
      this.desiredTarget.copy(this.focusTarget(
        this.focusNodePosition,
        this.desiredYaw,
        this.desiredPitch,
        this.desiredDistance,
      ))
    }
  }

  private updateLabelDeclutter() {
    const width = Math.max(1, this.container.clientWidth)
    const height = Math.max(1, this.container.clientHeight)
    const cameraRight = new THREE.Vector3(1, 0, 0).applyQuaternion(this.camera.quaternion)
    const cameraUp = new THREE.Vector3(0, 1, 0).applyQuaternion(this.camera.quaternion)
    const labelCenter = new THREE.Vector3()
    const projected = new THREE.Vector3()
    const candidates: LabelCandidate[] = []
    const hideOverviewLabels =
      this.fitDistance / Math.max(this.distance, 1) < LABEL_ZOOM_HIDE_THRESHOLD

    for (const [id, view] of this.nodeViews) {
      if (view.opacity < 0.05 || !view.group.visible) {
        view.targetLabelOpacity = 0
        continue
      }
      if (hideOverviewLabels && id !== this.hoveredId) {
        view.targetLabelOpacity = 0
        continue
      }
      view.label.getWorldPosition(labelCenter)
      // Labels are anchored at their lower edge so they grow upward from the
      // node. Collision rectangles need the visual center instead.
      labelCenter.addScaledVector(cameraUp, view.label.scale.y * 0.5)
      const towardLabel = labelCenter.clone().sub(this.camera.position)
      const cameraForward = new THREE.Vector3(0, 0, -1).applyQuaternion(this.camera.quaternion)
      if (towardLabel.dot(cameraForward) <= 0.1) {
        view.targetLabelOpacity = 0
        continue
      }

      const halfWidth = view.label.scale.x * 0.5
      const halfHeight = view.label.scale.y * cameraUp.length() * 0.5
      const points = [
        labelCenter.clone()
          .addScaledVector(cameraRight, -halfWidth)
          .addScaledVector(cameraUp, halfHeight),
        labelCenter.clone()
          .addScaledVector(cameraRight, halfWidth)
          .addScaledVector(cameraUp, halfHeight),
        labelCenter.clone()
          .addScaledVector(cameraRight, halfWidth)
          .addScaledVector(cameraUp, -halfHeight),
        labelCenter.clone()
          .addScaledVector(cameraRight, -halfWidth)
          .addScaledVector(cameraUp, -halfHeight),
      ].map((point) => {
        projected.copy(point).project(this.camera)
        return {
          x: (projected.x * 0.5 + 0.5) * width,
          y: (-projected.y * 0.5 + 0.5) * height,
        }
      })
      const left = Math.min(...points.map((point) => point.x))
      const right = Math.max(...points.map((point) => point.x))
      const top = Math.min(...points.map((point) => point.y))
      const bottom = Math.max(...points.map((point) => point.y))
      const priority = id === this.hoveredId
        ? 0
        : id === this.selectedId
          ? 1
          : id === VIRTUAL_ROOT_ID
            ? 2
            : this.focusedIds.has(id)
              ? 3
              : 4
      candidates.push({
        id,
        view,
        priority,
        left,
        right,
        top,
        bottom,
        readable: bottom - top >= 12,
      })
    }

    candidates.sort((a, b) => a.priority - b.priority || a.top - b.top)
    const occupied: LabelCandidate[] = []
    for (const candidate of candidates) {
      const mustShow = candidate.id === this.hoveredId ||
        candidate.id === this.selectedId ||
        candidate.id === VIRTUAL_ROOT_ID
      const offscreen = candidate.right < 0 ||
        candidate.left > width ||
        candidate.bottom < 0 ||
        candidate.top > height
      const collides = occupied.some((other) =>
        candidate.left < other.right + LABEL_GAP &&
        candidate.right + LABEL_GAP > other.left &&
        candidate.top < other.bottom + LABEL_GAP &&
        candidate.bottom + LABEL_GAP > other.top,
      )
      const visible = candidate.id === this.hoveredId ||
        (!offscreen && (mustShow || (candidate.readable && !collides)))
      candidate.view.targetLabelOpacity = visible
        ? candidate.view.opacity * candidate.view.labelEmphasis
        : 0
      if (visible) occupied.push(candidate)
    }
  }

  private updateHoverParticles(
    view: NodeVisual,
    now: number,
    hoverAmount: number,
  ) {
    if (hoverAmount <= 0.001) {
      view.particles.material.opacity = 0
      view.particles.visible = false
      return
    }
    const position = view.particles.geometry.getAttribute('position') as THREE.BufferAttribute
    const positions = position.array as Float32Array
    for (let index = 0; index < HOVER_PARTICLE_COUNT; index++) {
      const seedOffset = index * HOVER_PARTICLE_STRIDE
      const positionOffset = index * 3
      const angle = view.particleSeeds[seedOffset]
      const orbitFactor = view.particleSeeds[seedOffset + 1]
      const speed = view.particleSeeds[seedOffset + 2]
      const phase = view.particleSeeds[seedOffset + 3]
      const rise = view.particleSeeds[seedOffset + 4]
      const progress = (phase + now * 0.0001 * speed) % 1
      const radius = view.radius * orbitFactor * (1 - progress * 0.24)
      positions[positionOffset] = Math.cos(angle + progress * 1.15) * radius
      positions[positionOffset + 1] = 0.18 + progress * rise +
        Math.sin((progress + phase) * Math.PI * 2) * 0.07
      positions[positionOffset + 2] = Math.sin(angle + progress * 1.15) * radius
    }
    position.needsUpdate = true
    view.particles.material.opacity = hoverAmount * 0.86
    view.particles.material.size = view.radius * (0.24 + hoverAmount * 0.1)
    view.particles.visible = hoverAmount > 0.01
  }

  private animate = () => {
    if (this.disposed) return
    this.frameId = requestAnimationFrame(this.animate)
    const now = performance.now()
    const delta = Math.min(0.05, (now - this.lastFrame) / 1000)
    this.lastFrame = now
    const damping = 1 - Math.exp(-delta * 5.2)
    const opacityDamping = 1 - Math.exp(-delta * 8)
    const hoverDamping = 1 - Math.exp(-delta * HOVER_FADE_SPEED)

    this.smoothPointerX += (this.pointerX - this.smoothPointerX) * damping
    this.smoothPointerY += (this.pointerY - this.smoothPointerY) * damping
    const pointerInfluence = this.focus ? 0.2 : 1
    this.world.rotation.z = -this.smoothPointerX * 0.013 * pointerInfluence
    this.world.rotation.x = this.smoothPointerY * 0.01 * pointerInfluence
    if (!this.focus) this.world.rotation.y += delta * LAYOUT_ROTATION_SPEED
    this.dust.rotation.y += delta * 0.004

    this.viewTarget.lerp(this.desiredTarget, damping)
    if (this.introActive) {
      const elapsed = now - this.introStartedAt
      const progress = THREE.MathUtils.clamp(elapsed / INTRO_DURATION_MS, 0, 1)
      const eased = 1 - Math.pow(1 - progress, 3)
      this.distance = THREE.MathUtils.lerp(
        this.introFromDistance,
        this.introToDistance,
        eased,
      )
      this.pitch = THREE.MathUtils.lerp(
        this.introFromPitch,
        this.introToPitch,
        eased,
      )
      this.yaw = THREE.MathUtils.lerp(
        this.introFromYaw,
        this.introToYaw,
        eased,
      )
      if (progress >= 1) this.introActive = false
    } else {
      this.distance += (this.desiredDistance - this.distance) * damping
      this.pitch += (this.desiredPitch - this.pitch) * damping
      this.yaw += (this.desiredYaw - this.yaw) * damping
    }
    const horizontalDistance = Math.cos(this.pitch) * this.distance
    const cameraOffset = new THREE.Vector3(
      Math.sin(this.yaw) * horizontalDistance,
      Math.sin(this.pitch) * this.distance,
      Math.cos(this.yaw) * horizontalDistance,
    )
    this.camera.position.copy(this.viewTarget).add(cameraOffset)
    this.camera.lookAt(this.viewTarget)
    this.camera.updateMatrixWorld()
    this.updateHoverFromPointer()

    for (const [id, view] of this.nodeViews) {
      const hovered = id === this.hoveredId
      const selected = id === this.selectedId
      view.group.position.lerp(view.targetPosition, damping)
      view.label.position.lerp(view.labelTargetPosition, damping)
      view.opacity += (view.targetOpacity - view.opacity) * opacityDamping
      view.hoverAmount += ((hovered ? 1 : 0) - view.hoverAmount) * hoverDamping
      const neighborTitle = Boolean(
        this.focus && !selected && this.focusedIds.has(id),
      )
      const targetLabelScale = selected
        ? 1.1
        : hovered
          ? 1.08
          : neighborTitle
            ? 0.9
            : 1
      view.labelScale += (targetLabelScale - view.labelScale) * damping
      const targetLabelEmphasis = hovered ? 1 : neighborTitle ? 0.76 : 1
      view.labelEmphasis += (targetLabelEmphasis - view.labelEmphasis) * opacityDamping

      const materialOpacity = view.opacity
      const hoverAmount = view.hoverAmount
      view.plate.material.opacity = materialOpacity
      view.cap.material.opacity = materialOpacity
      view.crystal.material.opacity = materialOpacity
      const plateBase = selected
        ? Math.max(view.baseEmissive.plate, 0.68)
        : view.baseEmissive.plate
      const capBase = selected
        ? Math.max(view.baseEmissive.cap, 0.78)
        : view.baseEmissive.cap
      const crystalBase = selected
        ? Math.max(view.baseEmissive.crystal, 1.45)
        : view.baseEmissive.crystal
      view.plate.material.emissiveIntensity = THREE.MathUtils.lerp(
        plateBase,
        Math.max(plateBase + 0.58, 1.18),
        hoverAmount,
      )
      view.cap.material.emissiveIntensity = THREE.MathUtils.lerp(
        capBase,
        Math.max(capBase + 0.5, 1.18),
        hoverAmount,
      )
      view.crystal.material.emissiveIntensity = THREE.MathUtils.lerp(
        crystalBase,
        Math.max(crystalBase + 0.68, 2.25),
        hoverAmount,
      )
      const innerRingBase = materialOpacity * (selected ? 0.9 : 0.52)
      const outerRingBase = materialOpacity * (selected ? 0.95 : 0.48)
      view.innerRing.material.opacity = THREE.MathUtils.lerp(
        innerRingBase,
        materialOpacity,
        hoverAmount * 0.72,
      )
      view.outerRing.material.opacity = THREE.MathUtils.lerp(
        outerRingBase,
        materialOpacity,
        hoverAmount * 0.82,
      )
      const glowBase = materialOpacity * (selected ? 0.95 : 0.38)
      view.glow.material.opacity = THREE.MathUtils.lerp(
        glowBase,
        materialOpacity,
        hoverAmount,
      )
      const zoom = THREE.MathUtils.clamp(
        this.fitDistance / Math.max(this.distance, 1),
        0.45,
        5,
      )
      const labelScreenHeight = THREE.MathUtils.clamp(
        48 * Math.pow(zoom, 0.32),
        46,
        62,
      )
      const viewportHeight = Math.max(1, this.container.clientHeight)
      const nodeDistance = Math.max(
        1,
        this.camera.position.distanceTo(view.group.position),
      )
      const worldPerPixel = 2 * nodeDistance *
        Math.tan(THREE.MathUtils.degToRad(this.camera.fov / 2)) /
        viewportHeight
      const rootScale = id === VIRTUAL_ROOT_ID ? 1.08 : 1
      const labelWorldHeight = labelScreenHeight * worldPerPixel * view.labelScale * rootScale
      view.label.scale.set(
        labelWorldHeight * LABEL_ASPECT_RATIO,
        labelWorldHeight,
        1,
      )
      const pulse = (selected ? 1.08 + Math.sin(now * 0.004) * 0.035 : 1) *
        (1 + hoverAmount * 0.055)
      view.plate.scale.setScalar(view.radius * pulse)
      view.plate.scale.y = (id === VIRTUAL_ROOT_ID ? 1.5 : 1) * pulse
      view.cap.scale.setScalar(view.radius * pulse)
      view.cap.scale.y = (id === VIRTUAL_ROOT_ID ? 1.45 : 1) * pulse
      view.crystal.rotation.y += delta * (id === this.selectedId ? 0.75 : 0.24)
      view.crystal.scale.setScalar(view.radius * (id === VIRTUAL_ROOT_ID ? 1.18 : 0.9) * pulse)
      view.innerRing.scale.setScalar(view.radius * pulse)
      view.outerRing.rotation.z += delta * (id === this.selectedId ? 0.35 : 0.09)
      view.outerRing.scale.setScalar(view.radius * pulse)
      view.glow.scale.setScalar(
        view.radius * (id === VIRTUAL_ROOT_ID ? 4.8 : 4.1) *
        (1 + hoverAmount * 0.2),
      )
      this.updateHoverParticles(view, now, hoverAmount)
      view.group.visible = view.opacity > 0.01
    }
    for (const edge of this.edgeViews.values()) {
      const material = edge.line.material
      material.opacity += (edge.targetOpacity - material.opacity) * opacityDamping
      edge.line.visible = material.opacity > 0.01
    }

    this.updateLabelDeclutter()
    for (const view of this.nodeViews.values()) {
      view.labelOpacity += (view.targetLabelOpacity - view.labelOpacity) * opacityDamping
      view.label.material.opacity = view.labelOpacity
    }

    const selected = this.focus ? this.nodeViews.get(this.focus.node.id) : null
    if (selected?.group.visible) {
      selected.group.updateWorldMatrix(true, false)
      const nodeWorldPosition = new THREE.Vector3()
      selected.group.getWorldPosition(nodeWorldPosition)
      this.commentLayer.update(delta, this.camera, nodeWorldPosition, true)
    } else {
      this.commentLayer.update(delta, this.camera, new THREE.Vector3(), false)
    }
    this.renderer.render(this.scene, this.camera)
  }
}

function nearestAngle(current: number, target: number): number {
  const turn = Math.PI * 2
  return current + THREE.MathUtils.euclideanModulo(target - current + Math.PI, turn) - Math.PI
}
