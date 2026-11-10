import type * as THREE from "three";
import { CSS3DObject } from "three/addons/renderers/CSS3DRenderer.js";

import type { ComposerPose } from "../CommentLayer";
import { PROGRESS_CARD_WIDTH } from "../progressLayout";

export class ComposerLayer {
  private readonly scene: THREE.Scene;
  private object: CSS3DObject | null = null;
  private element: HTMLElement | null = null;

  constructor(scene: THREE.Scene) {
    this.scene = scene;
  }

  public attach(element: HTMLElement | null) {
    if (this.element === element) {
      return;
    }

    if (this.object) {
      this.scene.remove(this.object);
      this.object = null;
    }

    this.element = element;
    if (!element) {
      return;
    }

    const object = new CSS3DObject(element);
    const width = Math.max(1, element.offsetWidth || 620);

    object.scale.setScalar(PROGRESS_CARD_WIDTH / width);
    object.visible = false;
    object.element.style.pointerEvents = "auto";
    object.element.style.userSelect = "text";
    object.element.style.touchAction = "auto";
    object.element.style.visibility = "visible";

    this.object = object;
    this.scene.add(object);
  }

  public setVisible(visible: boolean) {
    if (this.object) {
      this.object.visible = visible;
    }
  }

  public isPointInside(clientX: number, clientY: number): boolean {
    const element = this.element;
    if (!this.object?.visible || !element) {
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

  public focusTextarea() {
    const textarea = this.element?.querySelector("textarea");
    if (!(textarea instanceof HTMLTextAreaElement)) {
      return;
    }

    textarea.focus({ preventScroll: true });
    const position = textarea.value.length;
    textarea.setSelectionRange(position, position);
  }

  public update(pose: ComposerPose) {
    if (!this.object) {
      return;
    }

    this.object.position.copy(pose.position);
    this.object.rotation.set(0, pose.rotationY, 0);
    this.object.visible = pose.opacity > 0.01;
    this.object.element.style.opacity = String(
      Math.min(1, pose.opacity * 1.08),
    );
    this.object.element.style.pointerEvents =
      pose.opacity > 0.01 ? "auto" : "none";
  }

  public dispose() {
    this.attach(null);
  }
}
