import * as THREE from "three";
import { CSS3DRenderer } from "three/addons/renderers/CSS3DRenderer.js";

import type { ProgressItem } from "@/api";
import type { ThemeMode } from "@/composables/useTheme";
import type { GraphNeighborhood, GraphNode } from "@/graph/types";

import { CommentLayer } from "./CommentLayer";
import type { GraphLayout } from "./layout";
import { VIRTUAL_ROOT_ID, layoutGraph } from "./layout";
import { CameraRig } from "./scene/cameraRig";
import { ComposerLayer } from "./scene/composerLayer";
import { EdgeLayer } from "./scene/edgeLayer";
import { GraphEnvironment } from "./scene/environment";
import { NodeLayer } from "./scene/nodeLayer";
import { PointerController } from "./scene/pointerController";
import { WorldMotion } from "./scene/worldMotion";
import { buildVirtualRootNode } from "./virtualRoot";

interface SceneCallbacks {
  onBlank: () => void;
  onProgressNeedOlder: () => void;
  onProgressOpen: (key: string) => void;
  onReaction: (commentId: number, emoji: string) => void;
  onSelect: (id: number) => void;
}

export class GraphScene {
  private readonly container: HTMLElement;
  private readonly callbacks: SceneCallbacks;
  private readonly scene = new THREE.Scene();
  private readonly renderer: THREE.WebGLRenderer;
  private readonly cssRenderer = new CSS3DRenderer();
  private readonly world = new THREE.Group();
  private readonly dataNodes = new Map<number, GraphNode>();
  private readonly extraNodes = new Map<number, GraphNode>();
  private readonly commentLayer = new CommentLayer();
  private readonly environment: GraphEnvironment;
  private readonly cameraRig = new CameraRig();
  private readonly worldMotion: WorldMotion;
  private readonly nodeLayer: NodeLayer;
  private readonly edgeLayer: EdgeLayer;
  private readonly composerLayer: ComposerLayer;
  private readonly pointerController: PointerController;
  private readonly resizeObserver: ResizeObserver;
  private readonly zeroVector = new THREE.Vector3();
  private layout: GraphLayout = layoutGraph([]);
  private focus: GraphNeighborhood | null = null;
  private focusedIds = new Set<number>();
  private selectedId: number | null = null;
  private hoveredId: number | null = null;
  private commentNodeId: number | null = null;
  private hasInteracted = false;
  private frameId = 0;
  private lastFrame = performance.now();
  private disposed = false;

  constructor(
    container: HTMLElement,
    callbacks: SceneCallbacks,
    theme: ThemeMode = "dark",
  ) {
    this.container = container;
    this.callbacks = callbacks;
    this.renderer = new THREE.WebGLRenderer({
      antialias: true,
      alpha: false,
      powerPreference: "high-performance",
    });

    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    this.renderer.outputColorSpace = THREE.SRGBColorSpace;
    this.renderer.toneMapping = THREE.ACESFilmicToneMapping;
    this.renderer.toneMappingExposure = theme === "light" ? 0.96 : 1.1;
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

    this.scene.add(this.world, this.commentLayer.group);
    this.commentLayer.setTheme(theme);
    this.environment = new GraphEnvironment(this.scene, this.world, theme);
    this.worldMotion = new WorldMotion(this.world);
    this.nodeLayer = new NodeLayer(this.world, theme);
    this.edgeLayer = new EdgeLayer(this.world, theme);
    this.composerLayer = new ComposerLayer(this.scene);
    this.pointerController = new PointerController(
      this.renderer.domElement,
      this.cameraRig.camera,
      this.nodeLayer,
      this.composerLayer,
      {
        canPan: () => !this.focus,
        focusComposer: () => this.composerLayer.focusTextarea(),
        onBlank: () => this.callbacks.onBlank(),
        onHoverChange: (id) => {
          this.hoveredId = id;
          this.syncVisualState();
        },
        onInteract: () => {
          this.hasInteracted = true;
        },
        onPan: (dx, dy, viewportHeight) =>
          this.cameraRig.panByPixels(dx, dy, viewportHeight),
        onPointerGestureStart: () => this.cameraRig.stopIntro(),
        onProgressOpen: (key) => this.callbacks.onProgressOpen(key),
        onReaction: (commentId, emoji) =>
          this.callbacks.onReaction(commentId, emoji),
        onSelect: (id) => this.callbacks.onSelect(id),
        onZoom: (deltaY) => this.cameraRig.zoom(deltaY),
        pickProgress: (raycaster) =>
          this.commentLayer.pickProgressAt(raycaster),
      },
    );

    this.resizeObserver = new ResizeObserver(() => this.resize());
    this.resizeObserver.observe(container);
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
    if (!this.hasInteracted && !this.focus) {
      this.fitView();
    }
  }

  public setTheme(theme: ThemeMode) {
    this.environment.setTheme(theme);
    this.renderer.toneMappingExposure = theme === "light" ? 0.96 : 1.1;
    this.nodeLayer.setTheme(theme);
    this.edgeLayer.setTheme(theme);
    this.commentLayer.setTheme(theme);
  }

  public setProgress(items: ProgressItem[], loading = false) {
    const nodeId =
      items[0]?.kind === "comment"
        ? items[0].comment.nodeId
        : items[0]?.descriptionChange.nodeId;

    this.commentNodeId = nodeId ?? this.focus?.node.id ?? null;
    this.commentLayer.setProgress(items, loading);
    this.refreshLabelTargets();
  }

  public setCommentComposerVisible(visible: boolean) {
    this.commentLayer.setComposerVisible(visible);
    this.composerLayer.setVisible(visible);
  }

  public attachCommentComposer(element: HTMLElement | null) {
    this.composerLayer.attach(element);
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

    this.cameraRig.stopIntro();

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

    const captureSnapshot = !this.focus;
    const virtual = neighborhood.node.id === VIRTUAL_ROOT_ID;

    this.focus = neighborhood;
    this.selectedId = neighborhood.node.id;
    this.commentNodeId = neighborhood.node.id;
    this.hoveredId = null;
    this.focusedIds = ids;
    this.commentLayer.setEmptyEnabled(!virtual);
    this.pointerController.clearHover();
    this.world.updateMatrixWorld(true);

    const focusPosition = new THREE.Vector3();

    if (!this.nodeLayer.getWorldPosition(this.selectedId, focusPosition)) {
      focusPosition.set(0, 0.9, 0);
    }

    this.cameraRig.focus(
      focusPosition,
      virtual,
      this.layout.radius,
      captureSnapshot,
    );

    if (virtual) {
      this.worldMotion.startVirtualFocus(performance.now());
    }

    this.hasInteracted = true;
    this.rebuildViews();
  }

  public clearFocus() {
    if (!this.focus) {
      return;
    }

    this.focus = null;
    this.selectedId = null;
    this.hoveredId = null;
    this.focusedIds.clear();
    this.pointerController.clearHover();
    this.cameraRig.clearFocus();
    this.hasInteracted = true;
    this.refreshLabelTargets();
    this.syncVisualState();
  }

  public fitView() {
    if (this.dataNodes.size === 0) {
      return;
    }

    this.cameraRig.fitView(this.layout.radius);
  }

  public destroy() {
    this.disposed = true;
    cancelAnimationFrame(this.frameId);
    this.resizeObserver.disconnect();
    this.pointerController.dispose();
    this.nodeLayer.dispose();
    this.edgeLayer.dispose();
    this.composerLayer.dispose();
    this.commentLayer.dispose();
    this.environment.dispose();
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
      nodes.push(
        buildVirtualRootNode(
          nodes.filter((node) => node.parentId == null).length,
        ),
      );
    }

    this.layout = layoutGraph(nodes);
    this.nodeLayer.sync(nodes, this.layout);
    this.refreshLabelTargets();
    this.edgeLayer.sync(this.layout);
    this.syncVisualState();
  }

  private refreshLabelTargets() {
    this.nodeLayer.refreshLabelTargets(
      this.cameraRig.targetYaw,
      this.worldMotion.yaw,
    );
  }

  private syncVisualState() {
    this.nodeLayer.setState({
      commentNodeId: this.commentNodeId,
      focusActive: this.focus != null,
      focusedIds: this.focusedIds,
      hoveredId: this.hoveredId,
      selectedId: this.selectedId,
    });
    this.nodeLayer.applyVisualState();
    this.edgeLayer.setState({
      focusActive: this.focus != null,
      focusedIds: this.focusedIds,
      selectedId: this.selectedId,
    });
    this.edgeLayer.applyVisualState();
  }

  private resize() {
    const width = Math.max(1, this.container.clientWidth);
    const height = Math.max(1, this.container.clientHeight);

    this.cameraRig.resize(width, height);
    this.renderer.setSize(width, height, false);
    this.cssRenderer.setSize(width, height);
  }

  private animate = () => {
    if (this.disposed) {
      return;
    }

    this.frameId = requestAnimationFrame(this.animate);

    const now = performance.now();
    const delta = Math.min(0.05, (now - this.lastFrame) / 1000);

    this.lastFrame = now;
    this.pointerController.updateSmoothing(delta);
    this.worldMotion.update(delta, now, this.cameraRig.camera, {
      focusActive: this.focus != null,
      pointerX: this.pointerController.tiltX,
      pointerY: this.pointerController.tiltY,
      selectedVirtual: this.selectedId === VIRTUAL_ROOT_ID,
    });
    this.environment.update(delta);
    this.cameraRig.update(delta, now);
    this.pointerController.updateHover();

    const width = Math.max(1, this.container.clientWidth);
    const height = Math.max(1, this.container.clientHeight);

    this.nodeLayer.update(delta, {
      camera: this.cameraRig.camera,
      cameraDistance: this.cameraRig.currentDistance,
      delta,
      fitDistance: this.cameraRig.fittedDistance,
      now,
      viewportHeight: height,
      viewportWidth: width,
    });
    this.edgeLayer.update(delta);

    const selectedNodeId = this.focus?.node.id;
    const selectedWorldPosition = new THREE.Vector3();
    const commentTargetVisible =
      selectedNodeId != null &&
      this.nodeLayer.getVisualWorldPosition(
        selectedNodeId,
        selectedWorldPosition,
      );

    this.commentLayer.update(
      delta,
      this.cameraRig.camera,
      commentTargetVisible ? selectedWorldPosition : this.zeroVector,
      commentTargetVisible,
    );
    this.composerLayer.update(this.commentLayer.composerPose);
    this.renderer.render(this.scene, this.cameraRig.camera);
    this.cssRenderer.render(this.scene, this.cameraRig.camera);
  };
}
