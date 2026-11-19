import * as THREE from "three";
import {
  CSS3DObject,
  CSS3DRenderer,
} from "three/addons/renderers/CSS3DRenderer.js";

import type { ThemeMode } from "@/composables/useTheme";
import type {
  GraphNeighborhood,
  GraphNode,
  SceneCallbacks,
} from "@/graph/types";
import type { ProgressItem } from "@/lib/api";

import { CameraRig } from "./CameraRig";
import { COMMENT_CARD_WIDTH, CommentLayer } from "./CommentLayer";
import { NodeLayer } from "./NodeLayer";
import type { GraphLayout } from "./layout";
import { VIRTUAL_ROOT_ID, layoutGraph } from "./layout";
import { SCENE_PALETTES } from "./palette";
import type { GroundVisual } from "./visuals";
import { createDust, createGround, updateGroundPalette } from "./visuals";

const LAYOUT_ROTATION_SPEED = THREE.MathUtils.degToRad(0.45);
const VIRTUAL_ROOT_ROTATION_SPEED = (Math.PI * 2) / 80;
const VIRTUAL_ROOT_ROTATION_DELAY_MS = 800;
const ROTATION_SPEED_RESPONSE = 2.4;
const HOVER_FADE_SPEED = 9;
const GRAVITY_ROLL_STRENGTH = 0.034;
const GRAVITY_PITCH_STRENGTH = 0.027;

export class GraphScene {
  private readonly container: HTMLElement;
  private readonly callbacks: SceneCallbacks;
  private readonly scene = new THREE.Scene();
  private readonly camera = new THREE.PerspectiveCamera(44, 1, 0.1, 1800);
  private readonly renderer: THREE.WebGLRenderer;
  private readonly cssRenderer = new CSS3DRenderer();
  private readonly world = new THREE.Group();
  private readonly cameraRig: CameraRig;
  private readonly nodeLayer: NodeLayer;
  private readonly commentLayer = new CommentLayer();
  private readonly ground: GroundVisual;
  private readonly dust: THREE.Points<
    THREE.BufferGeometry,
    THREE.PointsMaterial
  >;

  private readonly resizeObserver: ResizeObserver;
  private readonly dataNodes = new Map<number, GraphNode>();
  private readonly extraNodes = new Map<number, GraphNode>();
  private readonly raycaster = new THREE.Raycaster();
  private readonly pointerNdc = new THREE.Vector2();
  private readonly pointerClient = new THREE.Vector2();
  private readonly lastPointer = new THREE.Vector2();

  private theme: ThemeMode;
  private composerObject: CSS3DObject | null = null;
  private composerElement: HTMLElement | null = null;
  private layout: GraphLayout = layoutGraph([]);
  private focus: GraphNeighborhood | null = null;
  private focusedIds = new Set<number>();
  private selectedId: number | null = null;
  private hoveredId: number | null = null;
  private commentNodeId: number | null = null;
  private layoutYaw = 0;
  private layoutRotationSpeed = LAYOUT_ROTATION_SPEED;
  private virtualFocusStartedAt = 0;
  private pointerX = 0;
  private pointerY = 0;
  private pointerInside = false;
  private smoothPointerX = 0;
  private smoothPointerY = 0;
  private dragging = false;
  private dragDistance = 0;
  private frameId = 0;
  private lastFrame = performance.now();
  private disposed = false;

  private get palette() {
    return SCENE_PALETTES[this.theme];
  }

  private get canNavigateCanvas(): boolean {
    return !this.focus || this.selectedId === VIRTUAL_ROOT_ID;
  }

  constructor(
    container: HTMLElement,
    callbacks: SceneCallbacks,
    theme: ThemeMode = "dark",
  ) {
    this.container = container;
    this.callbacks = callbacks;
    this.theme = theme;
    this.renderer = new THREE.WebGLRenderer({
      antialias: true,
      alpha: false,
      powerPreference: "high-performance",
    });
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    this.renderer.outputColorSpace = THREE.SRGBColorSpace;
    this.renderer.toneMapping = THREE.ACESFilmicToneMapping;
    this.renderer.toneMappingExposure = 1.1;
    this.renderer.domElement.style.display = "block";
    this.renderer.domElement.style.width = "100%";
    this.renderer.domElement.style.height = "100%";
    this.renderer.domElement.style.cursor = "grab";
    this.renderer.domElement.style.touchAction = "none";
    container.appendChild(this.renderer.domElement);
    this.cssRenderer.domElement.style.position = "absolute";
    this.cssRenderer.domElement.style.inset = "0";
    this.cssRenderer.domElement.style.zIndex = "1";
    this.cssRenderer.domElement.style.pointerEvents = "none";
    container.appendChild(this.cssRenderer.domElement);

    this.scene.background = new THREE.Color(this.palette.background);
    this.scene.fog = new THREE.FogExp2(this.palette.fog, 0.0024);
    this.ground = createGround(this.palette);
    this.world.add(this.ground.group);
    this.scene.add(this.world);
    this.scene.add(this.commentLayer.group);
    this.commentLayer.setTheme(this.theme);
    this.dust = createDust();
    this.scene.add(this.dust);
    this.scene.add(new THREE.HemisphereLight(0xff_e2_a7, 0x07_10_18, 2.2));
    const key = new THREE.DirectionalLight(0xff_e0_a0, 2.7);
    key.position.set(28, 60, 34);
    this.scene.add(key);
    const rim = new THREE.DirectionalLight(0x77_b9_d1, 1.2);
    rim.position.set(-36, 20, -28);
    this.scene.add(rim);

    this.cameraRig = new CameraRig(this.camera);
    this.nodeLayer = new NodeLayer(this.world);
    this.resizeObserver = new ResizeObserver(() => this.resize());
    this.resizeObserver.observe(container);
    this.bindEvents();
    this.resize();
    this.animate();
  }

  public setNodes(nodes: GraphNode[]) {
    this.dataNodes.clear();
    for (const node of nodes) {
      this.dataNodes.set(node.id, node);
    }
    for (const id of this.extraNodes.keys()) {
      if (id !== VIRTUAL_ROOT_ID && !this.dataNodes.has(id)) {
        this.extraNodes.delete(id);
      }
    }
    this.rebuildViews();
    if (!this.cameraRig.hasInteracted && !this.focus) {
      this.fitView();
    }
  }

  public setTheme(theme: ThemeMode) {
    this.theme = theme;
    this.scene.background = new THREE.Color(this.palette.background);
    if (this.scene.fog instanceof THREE.FogExp2) {
      this.scene.fog.color.setHex(this.palette.fog);
    }
    updateGroundPalette(this.ground, this.palette);
    this.dust.material.color.setHex(this.palette.dust);
    this.dust.material.opacity = this.palette.dustOpacity;
    this.renderer.toneMappingExposure = theme === "light" ? 0.96 : 1.1;
    this.nodeLayer.setTheme(theme);
    this.commentLayer.setTheme(theme);
  }

  public setProgress(items: ProgressItem[], loading = false) {
    const nodeId =
      items[0]?.kind === "comment"
        ? items[0].comment.nodeId
        : items[0]?.descriptionChange.nodeId;
    this.commentNodeId = nodeId ?? this.focus?.node.id ?? null;
    this.commentLayer.setProgress(items, loading);
    this.nodeLayer.updateLabelTargets(
      this.layout,
      this.selectedId,
      this.commentNodeId,
      this.cameraRig.getDesiredYaw(),
      this.layoutYaw,
    );
  }

  public setCommentComposerVisible(visible: boolean) {
    this.commentLayer.setComposerVisible(visible);
    if (this.composerObject) {
      this.composerObject.visible = visible;
    }
  }

  public attachCommentComposer(element: HTMLElement | null) {
    if (this.composerElement === element) {
      return;
    }
    if (this.composerObject) {
      this.scene.remove(this.composerObject);
      this.composerObject = null;
    }
    this.composerElement = element;
    if (!element) {
      return;
    }
    const object = new CSS3DObject(element);
    const width = Math.max(1, element.offsetWidth || 620);
    object.scale.setScalar(COMMENT_CARD_WIDTH / width);
    object.visible = false;
    object.element.style.pointerEvents = "auto";
    object.element.style.userSelect = "text";
    object.element.style.touchAction = "auto";
    object.element.style.visibility = "visible";
    this.composerObject = object;
    this.scene.add(object);
  }

  public scrollComments(delta: number) {
    if (!this.focus) {
      return;
    }
    if (this.commentLayer.scrollBy(delta)) {
      this.callbacks.onProgressNeedOlder();
    }
  }

  public setFocus(neighborhood: GraphNeighborhood | null) {
    if (!neighborhood) {
      this.clearFocus();

      return;
    }
    this.cameraRig.cancelIntro();

    const related = new Map<number, GraphNode>();
    related.set(neighborhood.node.id, neighborhood.node);
    if (neighborhood.parent) {
      related.set(neighborhood.parent.id, neighborhood.parent);
    }
    for (const child of neighborhood.children) {
      related.set(child.id, child);
    }

    // Keep the entire ancestry in view so the camera turn lands on a visible
    // path from the selected node back to the guild core.
    let parentId = neighborhood.node.parentId;
    const visited = new Set<number>([neighborhood.node.id]);
    while (parentId != null && !visited.has(parentId)) {
      visited.add(parentId);
      const ancestor = this.dataNodes.get(parentId);
      if (!ancestor) {
        break;
      }
      related.set(ancestor.id, ancestor);
      parentId = ancestor.parentId;
    }

    for (const node of related.values()) {
      this.extraNodes.set(node.id, node);
    }
    const ids = new Set(related.keys());
    ids.add(VIRTUAL_ROOT_ID);

    if (!this.focus) {
      this.cameraRig.saveSnapshot();
    }

    this.focus = neighborhood;
    this.selectedId = neighborhood.node.id;
    this.commentNodeId = neighborhood.node.id;
    this.commentLayer.setEmptyEnabled(neighborhood.node.id !== VIRTUAL_ROOT_ID);
    this.hoveredId = null;
    this.focusedIds = ids;
    this.world.updateMatrixWorld(true);
    const focusNodePosition = new THREE.Vector3(
      this.layout.positions.get(this.selectedId)?.x ?? 0,
      0.9,
      this.layout.positions.get(this.selectedId)?.z ?? 0,
    ).applyMatrix4(this.world.matrixWorld);

    // The camera sits beyond the selected node on its radial axis. Looking back
    // toward the center therefore places the selected archive in front and the
    // guild origin deep in the scene.
    if (neighborhood.node.id === VIRTUAL_ROOT_ID) {
      this.virtualFocusStartedAt = performance.now();
      this.cameraRig.focusVirtualRoot(
        this.camera,
        this.layout.radius,
        focusNodePosition,
      );
    } else {
      this.cameraRig.focusNode(this.camera, focusNodePosition);
    }
    this.rebuildViews();
    this.renderer.domElement.style.cursor = this.canNavigateCanvas
      ? "grab"
      : "default";
  }

  public clearFocus() {
    if (!this.focus) {
      return;
    }
    this.focus = null;
    this.selectedId = null;
    this.hoveredId = null;
    this.focusedIds.clear();
    this.cameraRig.clearFocus();
    this.nodeLayer.updateLabelTargets(
      this.layout,
      this.selectedId,
      this.commentNodeId,
      this.cameraRig.getDesiredYaw(),
      this.layoutYaw,
    );
    this.nodeLayer.applyVisualState(
      false,
      this.selectedId,
      this.hoveredId,
      this.focusedIds,
    );
    this.renderer.domElement.style.cursor = "grab";
  }

  public fitView() {
    if (this.dataNodes.size === 0) {
      return;
    }
    this.cameraRig.fitOverview(this.camera, this.layout.radius);
  }

  public destroy() {
    this.disposed = true;
    cancelAnimationFrame(this.frameId);
    this.resizeObserver.disconnect();
    this.unbindEvents();
    this.nodeLayer.dispose();
    this.commentLayer.dispose();
    if (this.composerObject) {
      this.scene.remove(this.composerObject);
      this.composerObject = null;
      this.composerElement = null;
    }
    this.ground.plane.geometry.dispose();
    this.ground.plane.material.dispose();
    this.ground.grid.geometry.dispose();
    this.ground.grid.material.dispose();
    this.dust.geometry.dispose();
    if (Array.isArray(this.dust.material)) {
      for (const material of this.dust.material) {
        material.dispose();
      }
    } else {
      this.dust.material.dispose();
    }
    this.renderer.dispose();
    this.renderer.domElement.remove();
    this.cssRenderer.domElement.remove();
  }

  private rebuildViews() {
    const nodes = [...this.extraNodes.values()];
    const known = new Set(nodes.map((node) => node.id));
    for (const node of this.dataNodes.values()) {
      if (!known.has(node.id)) {
        nodes.push(node);
      }
    }
    if (!known.has(VIRTUAL_ROOT_ID)) {
      nodes.push({
        id: VIRTUAL_ROOT_ID,
        parentId: null,
        title: "开拓者公会",
        sortOrder: -1,
        completedAt: null,
        childCount: nodes.filter((node) => node.parentId == null).length,
        commentCount: 0,
      });
    }
    this.layout = layoutGraph(nodes);
    this.nodeLayer.sync(nodes, this.layout, this.theme);
    this.nodeLayer.updateLabelTargets(
      this.layout,
      this.selectedId,
      this.commentNodeId,
      this.cameraRig.getDesiredYaw(),
      this.layoutYaw,
    );
    this.nodeLayer.applyVisualState(
      Boolean(this.focus),
      this.selectedId,
      this.hoveredId,
      this.focusedIds,
    );
  }

  private bindEvents() {
    this.renderer.domElement.addEventListener(
      "pointerdown",
      this.onPointerDown,
    );
    this.renderer.domElement.addEventListener(
      "pointermove",
      this.onPointerMove,
    );
    this.renderer.domElement.addEventListener("pointerup", this.onPointerUp);
    this.renderer.domElement.addEventListener(
      "pointercancel",
      this.onPointerUp,
    );
    this.renderer.domElement.addEventListener(
      "pointerleave",
      this.onPointerLeave,
    );
    this.renderer.domElement.addEventListener("wheel", this.onWheel, {
      passive: false,
    });
  }

  private unbindEvents() {
    this.renderer.domElement.removeEventListener(
      "pointerdown",
      this.onPointerDown,
    );
    this.renderer.domElement.removeEventListener(
      "pointermove",
      this.onPointerMove,
    );
    this.renderer.domElement.removeEventListener("pointerup", this.onPointerUp);
    this.renderer.domElement.removeEventListener(
      "pointercancel",
      this.onPointerUp,
    );
    this.renderer.domElement.removeEventListener(
      "pointerleave",
      this.onPointerLeave,
    );
    this.renderer.domElement.removeEventListener("wheel", this.onWheel);
  }

  private onPointerDown = (event: PointerEvent) => {
    if (event.button !== 0) {
      return;
    }
    if (this.isPointerOverComposer(event.clientX, event.clientY)) {
      event.preventDefault();
      event.stopPropagation();
      this.focusComposer();
      this.renderer.domElement.style.cursor = "text";

      return;
    }
    this.cameraRig.cancelIntro();
    this.dragging = true;
    this.pointerInside = true;
    this.pointerClient.set(event.clientX, event.clientY);
    this.hoveredId = null;
    this.nodeLayer.applyVisualState(
      Boolean(this.focus),
      this.selectedId,
      this.hoveredId,
      this.focusedIds,
    );
    this.dragDistance = 0;
    this.lastPointer.set(event.clientX, event.clientY);
    this.renderer.domElement.setPointerCapture(event.pointerId);
    this.renderer.domElement.style.cursor = this.canNavigateCanvas
      ? "grabbing"
      : "default";
  };

  private onPointerMove = (event: PointerEvent) => {
    const rect = this.renderer.domElement.getBoundingClientRect();
    this.pointerX = ((event.clientX - rect.left) / rect.width) * 2 - 1;
    this.pointerY = ((event.clientY - rect.top) / rect.height) * 2 - 1;
    this.pointerInside = true;
    this.pointerClient.set(event.clientX, event.clientY);
    if (!this.dragging) {
      this.updateHoverFromPointer();

      return;
    }
    const dx = event.clientX - this.lastPointer.x;
    const dy = event.clientY - this.lastPointer.y;
    this.dragDistance += Math.abs(dx) + Math.abs(dy);
    this.lastPointer.set(event.clientX, event.clientY);
    if (!this.canNavigateCanvas || this.dragDistance < 3) {
      return;
    }
    this.cameraRig.pan(this.camera, dx, dy, rect.height);
    this.renderer.domElement.style.cursor = "grabbing";
  };

  private onPointerUp = (event: PointerEvent) => {
    if (!this.dragging) {
      return;
    }
    this.dragging = false;
    if (this.renderer.domElement.hasPointerCapture(event.pointerId)) {
      this.renderer.domElement.releasePointerCapture(event.pointerId);
    }
    if (this.dragDistance > 5) {
      this.renderer.domElement.style.cursor = this.canNavigateCanvas
        ? "grab"
        : "default";

      return;
    }
    const progress = this.pickProgressAt(event.clientX, event.clientY);
    if (progress) {
      if (progress.emoji && progress.commentId != null) {
        this.callbacks.onReaction(progress.commentId, progress.emoji);
      } else {
        this.callbacks.onProgressOpen(progress.key);
      }
      this.renderer.domElement.style.cursor = "pointer";

      return;
    }
    const id = this.pickNodeAt(event.clientX, event.clientY, true);
    this.renderer.domElement.style.cursor =
      id == null ? (this.canNavigateCanvas ? "grab" : "default") : "pointer";
    if (id == null) {
      this.callbacks.onBlank();
    } else {
      this.callbacks.onSelect(id);
    }
  };

  private onPointerLeave = () => {
    if (this.dragging) {
      return;
    }
    this.pointerX = 0;
    this.pointerY = 0;
    this.pointerInside = false;
    this.hoveredId = null;
    this.nodeLayer.applyVisualState(
      Boolean(this.focus),
      this.selectedId,
      this.hoveredId,
      this.focusedIds,
    );
  };

  private onWheel = (event: WheelEvent) => {
    if (!this.canNavigateCanvas) {
      return;
    }
    event.preventDefault();
    this.cameraRig.zoom(event.deltaY);
  };

  private updateHoverFromPointer() {
    if (!this.pointerInside || this.dragging) {
      return;
    }
    if (
      this.isPointerOverComposer(this.pointerClient.x, this.pointerClient.y)
    ) {
      if (this.hoveredId !== null) {
        this.hoveredId = null;
        this.nodeLayer.applyVisualState(
          Boolean(this.focus),
          this.selectedId,
          this.hoveredId,
          this.focusedIds,
        );
      }
      this.renderer.domElement.style.cursor = "text";

      return;
    }
    const progress = this.pickProgressAt(
      this.pointerClient.x,
      this.pointerClient.y,
    );
    if (progress) {
      if (this.hoveredId !== null) {
        this.hoveredId = null;
        this.nodeLayer.applyVisualState(
          Boolean(this.focus),
          this.selectedId,
          this.hoveredId,
          this.focusedIds,
        );
      }
      this.renderer.domElement.style.cursor = "pointer";

      return;
    }
    const hovered = this.pickNodeAt(
      this.pointerClient.x,
      this.pointerClient.y,
      true,
    );
    const idleCursor = this.canNavigateCanvas ? "grab" : "default";
    if (hovered === this.hoveredId) {
      this.renderer.domElement.style.cursor =
        hovered == null ? idleCursor : "pointer";

      return;
    }
    this.hoveredId = hovered;
    this.nodeLayer.applyVisualState(
      Boolean(this.focus),
      this.selectedId,
      this.hoveredId,
      this.focusedIds,
    );
    this.renderer.domElement.style.cursor =
      hovered == null ? idleCursor : "pointer";
  }

  private setPointerNdc(clientX: number, clientY: number): DOMRect {
    const rect = this.renderer.domElement.getBoundingClientRect();
    this.pointerNdc.set(
      ((clientX - rect.left) / rect.width) * 2 - 1,
      -((clientY - rect.top) / rect.height) * 2 + 1,
    );

    return rect;
  }

  private pickProgressAt(clientX: number, clientY: number) {
    if (!this.focus) {
      return null;
    }
    this.setPointerNdc(clientX, clientY);
    this.raycaster.setFromCamera(this.pointerNdc, this.camera);

    return this.commentLayer.pickProgressAt(this.raycaster);
  }

  private isPointerOverComposer(clientX: number, clientY: number): boolean {
    const element = this.composerElement;
    if (!this.composerObject || !this.composerObject.visible || !element) {
      return false;
    }
    const rect = element.getBoundingClientRect();

    return (
      clientX >= rect.left &&
      clientX <= rect.right &&
      clientY >= rect.top &&
      clientY <= rect.bottom
    );
  }

  private focusComposer() {
    const textarea = this.composerElement?.querySelector("textarea");
    if (!(textarea instanceof HTMLTextAreaElement)) {
      return;
    }
    textarea.focus({ preventScroll: true });
    const position = textarea.value.length;
    textarea.setSelectionRange(position, position);
  }

  private pickNodeAt(
    clientX: number,
    clientY: number,
    includeVirtual = false,
  ): number | null {
    const rect = this.setPointerNdc(clientX, clientY);

    return this.nodeLayer.pickNodeAt(
      this.camera,
      this.pointerNdc,
      rect.height,
      includeVirtual,
    );
  }

  private resize() {
    const width = Math.max(1, this.container.clientWidth);
    const height = Math.max(1, this.container.clientHeight);
    this.camera.aspect = width / height;
    this.camera.updateProjectionMatrix();
    this.renderer.setSize(width, height, false);
    this.cssRenderer.setSize(width, height);
    this.cameraRig.onResize(
      this.camera,
      Boolean(this.focus),
      this.focus?.node.id === VIRTUAL_ROOT_ID,
    );
  }

  private animate = () => {
    if (this.disposed) {
      return;
    }
    this.frameId = requestAnimationFrame(this.animate);
    const now = performance.now();
    const delta = Math.min(0.05, (now - this.lastFrame) / 1000);
    this.lastFrame = now;
    const damping = 1 - Math.exp(-delta * 5.2);
    const opacityDamping = 1 - Math.exp(-delta * 8);
    const hoverDamping = 1 - Math.exp(-delta * HOVER_FADE_SPEED);

    this.smoothPointerX += (this.pointerX - this.smoothPointerX) * damping;
    this.smoothPointerY += (this.pointerY - this.smoothPointerY) * damping;
    const pointerInfluence = this.focus ? 0.5 : 1;
    const virtualRotationReady =
      this.selectedId === VIRTUAL_ROOT_ID &&
      now - this.virtualFocusStartedAt >= VIRTUAL_ROOT_ROTATION_DELAY_MS;
    const targetRotationSpeed = this.focus
      ? virtualRotationReady
        ? VIRTUAL_ROOT_ROTATION_SPEED
        : 0
      : LAYOUT_ROTATION_SPEED;
    const rotationSpeedDamping = 1 - Math.exp(-delta * ROTATION_SPEED_RESPONSE);
    this.layoutRotationSpeed +=
      (targetRotationSpeed - this.layoutRotationSpeed) * rotationSpeedDamping;
    this.layoutYaw += delta * this.layoutRotationSpeed;
    const layoutRotation = new THREE.Quaternion().setFromAxisAngle(
      new THREE.Vector3(0, 1, 0),
      this.layoutYaw,
    );
    const cameraForward = new THREE.Vector3(0, 0, -1)
      .applyQuaternion(this.camera.quaternion)
      .normalize();
    const cameraRight = new THREE.Vector3(1, 0, 0)
      .applyQuaternion(this.camera.quaternion)
      .normalize();
    const gravityTilt = new THREE.Quaternion().setFromAxisAngle(
      cameraForward,
      this.smoothPointerX * GRAVITY_ROLL_STRENGTH * pointerInfluence,
    );
    gravityTilt.multiply(
      new THREE.Quaternion().setFromAxisAngle(
        cameraRight,
        this.smoothPointerY * GRAVITY_PITCH_STRENGTH * pointerInfluence,
      ),
    );
    this.world.quaternion.copy(layoutRotation).premultiply(gravityTilt);
    this.dust.rotation.y += delta * 0.004;

    this.cameraRig.update(this.camera, now, damping, Boolean(this.focus));
    this.updateHoverFromPointer();

    this.nodeLayer.update({
      delta,
      now,
      camera: this.camera,
      viewportWidth: this.container.clientWidth,
      viewportHeight: this.container.clientHeight,
      layout: this.layout,
      zoomRatio: this.cameraRig.zoomRatio(),
      focusActive: Boolean(this.focus),
      selectedId: this.selectedId,
      hoveredId: this.hoveredId,
      focusedIds: this.focusedIds,
      damping,
      opacityDamping,
      hoverDamping,
    });

    const nodeWorldPosition = new THREE.Vector3();
    const hasSelectedPosition =
      this.focus != null &&
      this.nodeLayer.getNodeWorldPosition(
        this.focus.node.id,
        nodeWorldPosition,
      );
    this.commentLayer.update(
      delta,
      this.camera,
      nodeWorldPosition,
      hasSelectedPosition,
    );
    if (this.composerObject) {
      const pose = this.commentLayer.composerPose;
      this.composerObject.position.copy(pose.position);
      this.composerObject.rotation.set(0, pose.rotationY, 0);
      this.composerObject.visible = pose.opacity > 0.01;
      this.composerObject.element.style.opacity = String(
        Math.min(1, pose.opacity * 1.08),
      );
      this.composerObject.element.style.pointerEvents =
        pose.opacity > 0.01 ? "auto" : "none";
    }
    this.renderer.render(this.scene, this.camera);
    this.cssRenderer.render(this.scene, this.camera);
  };
}
