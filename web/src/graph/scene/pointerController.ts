import * as THREE from "three";

import type { ProgressCardHit } from "../CommentLayer";
import type { ComposerLayer } from "./composerLayer";
import type { NodeLayer } from "./nodeLayer";

interface PointerCallbacks {
  canPan: () => boolean;
  focusComposer: () => void;
  onBlank: () => void;
  onHoverChange: (id: number | null) => void;
  onInteract: () => void;
  onPan: (dx: number, dy: number, viewportHeight: number) => void;
  onPointerGestureStart: () => void;
  onProgressOpen: (key: string) => void;
  onReaction: (commentId: number, emoji: string) => void;
  onSelect: (id: number) => void;
  onZoom: (deltaY: number) => void;
  pickProgress: (raycaster: THREE.Raycaster) => ProgressCardHit | null;
}

export class PointerController {
  private readonly domElement: HTMLElement;
  private readonly camera: THREE.PerspectiveCamera;
  private readonly nodeLayer: NodeLayer;
  private readonly composerLayer: ComposerLayer;
  private readonly callbacks: PointerCallbacks;
  private readonly raycaster = new THREE.Raycaster();
  private readonly pointerNdc = new THREE.Vector2();
  private readonly pointerClient = new THREE.Vector2();
  private readonly lastPointer = new THREE.Vector2();
  private pointerX = 0;
  private pointerY = 0;
  private pointerInside = false;
  private smoothPointerX = 0;
  private smoothPointerY = 0;
  private dragging = false;
  private dragDistance = 0;
  private hoveredId: number | null = null;

  constructor(
    domElement: HTMLElement,
    camera: THREE.PerspectiveCamera,
    nodeLayer: NodeLayer,
    composerLayer: ComposerLayer,
    callbacks: PointerCallbacks,
  ) {
    this.domElement = domElement;
    this.camera = camera;
    this.nodeLayer = nodeLayer;
    this.composerLayer = composerLayer;
    this.callbacks = callbacks;

    this.domElement.addEventListener("pointerdown", this.onPointerDown);
    this.domElement.addEventListener("pointermove", this.onPointerMove);
    this.domElement.addEventListener("pointerup", this.onPointerUp);
    this.domElement.addEventListener("pointercancel", this.onPointerUp);
    this.domElement.addEventListener("pointerleave", this.onPointerLeave);
    this.domElement.addEventListener("wheel", this.onWheel, { passive: false });
  }

  public get tiltX() {
    return this.smoothPointerX;
  }

  public get tiltY() {
    return this.smoothPointerY;
  }

  public updateSmoothing(delta: number) {
    const damping = 1 - Math.exp(-delta * 5.2);

    this.smoothPointerX += (this.pointerX - this.smoothPointerX) * damping;
    this.smoothPointerY += (this.pointerY - this.smoothPointerY) * damping;
  }

  public updateHover() {
    if (!this.pointerInside || this.dragging) {
      return;
    }

    if (
      this.composerLayer.isPointInside(
        this.pointerClient.x,
        this.pointerClient.y,
      )
    ) {
      this.setHovered(null);
      this.domElement.style.cursor = "text";

      return;
    }

    const raycaster = this.rayAt(this.pointerClient.x, this.pointerClient.y);
    if (this.callbacks.pickProgress(raycaster)) {
      this.setHovered(null);
      this.domElement.style.cursor = "pointer";

      return;
    }

    const hovered = this.pickNodeAt(
      this.pointerClient.x,
      this.pointerClient.y,
      true,
    );

    this.setHovered(hovered);
    this.domElement.style.cursor = hovered == null ? "grab" : "pointer";
  }

  public clearHover() {
    this.setHovered(null);
  }

  public dispose() {
    this.domElement.removeEventListener("pointerdown", this.onPointerDown);
    this.domElement.removeEventListener("pointermove", this.onPointerMove);
    this.domElement.removeEventListener("pointerup", this.onPointerUp);
    this.domElement.removeEventListener("pointercancel", this.onPointerUp);
    this.domElement.removeEventListener("pointerleave", this.onPointerLeave);
    this.domElement.removeEventListener("wheel", this.onWheel);
  }

  private onPointerDown = (event: PointerEvent) => {
    if (event.button !== 0) {
      return;
    }

    if (this.composerLayer.isPointInside(event.clientX, event.clientY)) {
      event.preventDefault();
      event.stopPropagation();
      this.callbacks.focusComposer();
      this.domElement.style.cursor = "text";

      return;
    }

    this.callbacks.onPointerGestureStart();
    this.dragging = true;
    this.pointerInside = true;
    this.pointerClient.set(event.clientX, event.clientY);
    this.setHovered(null);
    this.dragDistance = 0;
    this.lastPointer.set(event.clientX, event.clientY);
    this.domElement.setPointerCapture(event.pointerId);
    this.domElement.style.cursor = "grabbing";
  };

  private onPointerMove = (event: PointerEvent) => {
    const rect = this.domElement.getBoundingClientRect();

    this.pointerX = ((event.clientX - rect.left) / rect.width) * 2 - 1;
    this.pointerY = ((event.clientY - rect.top) / rect.height) * 2 - 1;
    this.pointerInside = true;
    this.pointerClient.set(event.clientX, event.clientY);

    if (!this.dragging) {
      this.updateHover();

      return;
    }

    const dx = event.clientX - this.lastPointer.x;
    const dy = event.clientY - this.lastPointer.y;

    this.dragDistance += Math.abs(dx) + Math.abs(dy);
    this.lastPointer.set(event.clientX, event.clientY);

    if (!this.callbacks.canPan() || this.dragDistance < 3) {
      return;
    }

    this.callbacks.onInteract();
    this.callbacks.onPan(dx, dy, rect.height);
    this.domElement.style.cursor = "grabbing";
  };

  private onPointerUp = (event: PointerEvent) => {
    if (!this.dragging) {
      return;
    }

    this.dragging = false;
    if (this.domElement.hasPointerCapture(event.pointerId)) {
      this.domElement.releasePointerCapture(event.pointerId);
    }
    if (this.dragDistance > 5) {
      this.domElement.style.cursor = "grab";

      return;
    }

    const raycaster = this.rayAt(event.clientX, event.clientY);
    const progress = this.callbacks.pickProgress(raycaster);

    if (progress) {
      if (progress.emoji && progress.commentId != null) {
        this.callbacks.onReaction(progress.commentId, progress.emoji);
      } else {
        this.callbacks.onProgressOpen(progress.key);
      }
      this.domElement.style.cursor = "pointer";

      return;
    }

    const id = this.pickNodeAt(event.clientX, event.clientY, true);

    this.domElement.style.cursor = id == null ? "grab" : "pointer";
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
    this.setHovered(null);
  };

  private onWheel = (event: WheelEvent) => {
    if (!this.callbacks.canPan()) {
      return;
    }

    event.preventDefault();
    this.callbacks.onInteract();
    this.callbacks.onZoom(event.deltaY);
  };

  private setHovered(id: number | null) {
    if (this.hoveredId === id) {
      return;
    }

    this.hoveredId = id;
    this.callbacks.onHoverChange(id);
  }

  private pickNodeAt(
    clientX: number,
    clientY: number,
    includeVirtual: boolean,
  ): number | null {
    const raycaster = this.rayAt(clientX, clientY);
    const rect = this.domElement.getBoundingClientRect();

    return this.nodeLayer.pick(
      raycaster,
      this.camera,
      rect.height,
      includeVirtual,
    );
  }

  private rayAt(clientX: number, clientY: number): THREE.Raycaster {
    const rect = this.domElement.getBoundingClientRect();

    this.pointerNdc.set(
      ((clientX - rect.left) / rect.width) * 2 - 1,
      -((clientY - rect.top) / rect.height) * 2 + 1,
    );
    this.raycaster.setFromCamera(this.pointerNdc, this.camera);

    return this.raycaster;
  }
}
