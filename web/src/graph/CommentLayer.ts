import * as THREE from "three";

import {
  CARD_CANVAS_HEIGHT,
  CARD_CANVAS_WIDTH,
  REACTION_CELL_WIDTH,
  REACTION_GAP,
  REACTION_ROW_BOTTOM,
  REACTION_ROW_TOP,
  REACTION_START_X,
  createEmptyTexture,
  createProgressTexture,
} from "./commentTextures";
import { smoothstep } from "./math";
import { progressTextureKey } from "./progress";
import {
  PROGRESS_CARD_CENTER_X as CARD_CENTER_X,
  PROGRESS_CARD_HEIGHT as CARD_HEIGHT,
  PROGRESS_CARD_WIDTH as CARD_WIDTH,
} from "./progressLayout";

interface ProgressVisual {
  anchor: THREE.Group;
  card: THREE.Mesh<THREE.PlaneGeometry, THREE.MeshBasicMaterial>;
  dot: THREE.Mesh<THREE.SphereGeometry, THREE.MeshBasicMaterial>;
  line: THREE.Line<THREE.BufferGeometry, THREE.LineBasicMaterial>;
  textureKey: string;
  y: number;
  targetY: number;
  opacity: number;
  reveal: number;
  pulse: number;
  item: ProgressItem;
}

export interface ProgressCardHit {
  key: string;
  commentId: number | null;
  emoji: string | null;
}

export interface ComposerPose {
  position: THREE.Vector3;
  rotationY: number;
  opacity: number;
}

const BASE_Y = 2.62;
const SPACING = 2.85;
const FADE_START = 6.2;
const FADE_END = 15.5;

export class CommentLayer {
  public readonly group = new THREE.Group();
  public readonly composerPose: ComposerPose = {
    position: new THREE.Vector3(),
    rotationY: 0,
    opacity: 0,
  };

  private readonly cardGeometry = new THREE.PlaneGeometry(
    CARD_WIDTH,
    CARD_HEIGHT,
  );

  private readonly dotGeometry = new THREE.SphereGeometry(0.15, 16, 12);
  private readonly composerDot: THREE.Mesh<
    THREE.SphereGeometry,
    THREE.MeshBasicMaterial
  >;

  private readonly composerLine: THREE.Line<
    THREE.BufferGeometry,
    THREE.LineBasicMaterial
  >;

  private readonly emptyLabel: THREE.Sprite;
  private readonly visuals = new Map<string, ProgressVisual>();
  private targetScroll = 0;
  private scroll = 0;
  private lastNodePosition: THREE.Vector3 | null = null;
  private commentsLoading = false;
  private emptyEnabled = true;
  private emptyOpacity = 0;
  private composerVisible = false;
  private composerOpacity = 0;
  private composerY = BASE_Y;
  private theme: ThemeMode = "dark";

  constructor() {
    this.group.visible = false;
    this.group.renderOrder = 5;
    this.composerDot = new THREE.Mesh(
      this.dotGeometry,
      new THREE.MeshBasicMaterial({
        color: 0xe7_b3_4d,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        depthTest: false,
        blending: THREE.AdditiveBlending,
      }),
    );
    this.composerLine = new THREE.Line(
      new THREE.BufferGeometry().setFromPoints([
        new THREE.Vector3(0, 0, 0),
        new THREE.Vector3(0, 1, 0),
      ]),
      new THREE.LineBasicMaterial({
        color: 0xd8_a8_4c,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        depthTest: false,
      }),
    );
    this.emptyLabel = new THREE.Sprite(
      new THREE.SpriteMaterial({
        map: createEmptyTexture(this.theme),
        transparent: true,
        opacity: 0,
        depthWrite: false,
        depthTest: false,
        toneMapped: false,
      }),
    );
    this.emptyLabel.scale.set(6.2, 0.98, 1);
    this.emptyLabel.visible = false;
    this.group.add(this.composerLine, this.composerDot, this.emptyLabel);
  }

  public setProgress(items: ProgressItem[], loading = false) {
    this.commentsLoading = loading;
    const previousCount = this.visuals.size;
    const previousIDs = new Set(this.visuals.keys());
    const previousCommentIDs = [...previousIDs]
      .filter((key) => key.startsWith("comment:"))
      .map((key) => Number(key.slice("comment:".length)));
    const previousMaxCommentID =
      previousCommentIDs.length > 0
        ? Math.max(...previousCommentIDs)
        : Number.NEGATIVE_INFINITY;
    const active = new Set(items.map(progressKey));
    for (const [id, visual] of this.visuals) {
      if (active.has(id)) {
        continue;
      }
      this.group.remove(visual.anchor, visual.line);
      this.disposeVisual(visual);
      this.visuals.delete(id);
    }

    for (const [index, item] of items.entries()) {
      const id = progressKey(item);
      let visual = this.visuals.get(id);
      if (visual) {
        visual.item = item;
        const textureKey = progressTextureKey(item);
        if (visual.textureKey !== textureKey) {
          const previousTexture = visual.card.material.map;
          visual.card.material.map = createProgressTexture(item, this.theme);
          visual.card.material.needsUpdate = true;
          previousTexture?.dispose();
          visual.textureKey = textureKey;
          visual.dot.material.color.setHex(
            item.kind === "comment" && item.comment.reactions.length > 0
              ? 0x6d_e2_c2
              : 0xe7_b3_4d,
          );
          visual.pulse = 1;
        }
      } else {
        visual = this.createVisual(item);
        this.visuals.set(id, visual);
        this.group.add(visual.anchor, visual.line);
      }
      visual.targetY = this.progressY(items.length, index);
    }

    if (previousCount > 0 && items.length > previousCount) {
      const added = (items.length - previousCount) * SPACING;
      const hasNewProgress = items.some(
        (item) =>
          item.kind === "comment" &&
          !previousIDs.has(progressKey(item)) &&
          item.comment.id > previousMaxCommentID,
      );
      if (hasNewProgress) {
        this.targetScroll = 0;
      } else {
        this.scroll -= added;
        this.targetScroll -= added;
      }
    }
    this.clampScroll();
  }

  public setComposerVisible(visible: boolean) {
    if (this.composerVisible === visible) {
      return;
    }
    this.composerVisible = visible;
    const comments = [...this.visuals.values()];
    for (const [index, visual] of comments.entries()) {
      visual.targetY = this.progressY(comments.length, index);
    }
    if (visible) {
      this.scroll = 0;
      this.targetScroll = 0;
    }
    this.clampScroll();
  }

  public setTheme(theme: ThemeMode) {
    this.theme = theme;
    for (const visual of this.visuals.values()) {
      const previousTexture = visual.card.material.map;
      visual.card.material.map = createProgressTexture(visual.item, theme);
      visual.card.material.needsUpdate = true;
      previousTexture?.dispose();
    }
    const previousEmpty = this.emptyLabel.material.map;
    this.emptyLabel.material.map = createEmptyTexture(theme);
    this.emptyLabel.material.needsUpdate = true;
    previousEmpty?.dispose();
  }

  public setEmptyEnabled(enabled: boolean) {
    this.emptyEnabled = enabled;
    if (!enabled) {
      this.emptyOpacity = 0;
      this.emptyLabel.material.opacity = 0;
      this.emptyLabel.visible = false;
    }
  }

  public scrollBy(delta: number): boolean {
    this.targetScroll += delta * 0.022;
    this.clampScroll();

    return this.targetScroll <= this.minScroll() + 0.15;
  }

  public pickProgressAt(raycaster: THREE.Raycaster): ProgressCardHit | null {
    const cards = [...this.visuals.values()]
      .filter((visual) => visual.opacity > 0.04 && visual.card.visible)
      .map((visual) => visual.card);
    const hit = raycaster.intersectObjects(cards, false)[0];
    if (!hit?.uv) {
      return null;
    }
    const key = hit.object.userData.progressKey;
    const commentId = hit.object.userData.commentId;
    if (typeof key !== "string") {
      return null;
    }
    const resolvedCommentID = typeof commentId === "number" ? commentId : null;
    const x = hit.uv.x * CARD_CANVAS_WIDTH;
    const y = (1 - hit.uv.y) * CARD_CANVAS_HEIGHT;
    if (
      resolvedCommentID == null ||
      y < REACTION_ROW_TOP ||
      y > REACTION_ROW_BOTTOM
    ) {
      return { key, commentId: resolvedCommentID, emoji: null };
    }
    const slotWidth = REACTION_CELL_WIDTH + REACTION_GAP;
    const index = Math.floor((x - REACTION_START_X) / slotWidth);
    const withinSlot =
      (x - REACTION_START_X) % slotWidth <= REACTION_CELL_WIDTH;
    if (index < 0 || index >= reactionKeys.length || !withinSlot) {
      return { key, commentId: resolvedCommentID, emoji: null };
    }

    return {
      key,
      commentId: resolvedCommentID,
      emoji: reactionKeys[index] ?? null,
    };
  }

  public update(
    delta: number,
    camera: THREE.Camera,
    nodeWorldPosition: THREE.Vector3,
    selected: boolean,
  ) {
    if (selected) {
      this.lastNodePosition ??= new THREE.Vector3();
      this.lastNodePosition.copy(nodeWorldPosition);
    }
    if (!this.lastNodePosition) {
      this.group.visible = false;

      return;
    }
    const scrollDamping = 1 - Math.exp(-delta * 13);
    const positionDamping = 1 - Math.exp(-delta * 10);
    const opacityDamping = 1 - Math.exp(-delta * 9);
    const revealDamping = 1 - Math.exp(-delta * 11);
    this.scroll += (this.targetScroll - this.scroll) * scrollDamping;
    const cameraRight = new THREE.Vector3(1, 0, 0).applyQuaternion(
      camera.quaternion,
    );
    cameraRight.y = 0;
    const cardRotation =
      cameraRight.lengthSq() > 0.0001
        ? Math.atan2(-cameraRight.z, cameraRight.x)
        : 0;

    for (const visual of this.visuals.values()) {
      const y = visual.targetY + this.scroll;
      visual.y += (y - visual.y) * positionDamping;
      visual.anchor.position.set(
        this.lastNodePosition.x,
        this.lastNodePosition.y + y,
        this.lastNodePosition.z,
      );
      visual.anchor.rotation.set(0, cardRotation, 0);
      const positions = visual.line.geometry.getAttribute(
        "position",
      ) as THREE.BufferAttribute;
      positions.setXYZ(
        0,
        this.lastNodePosition.x,
        this.lastNodePosition.y,
        this.lastNodePosition.z,
      );
      positions.setXYZ(
        1,
        this.lastNodePosition.x,
        this.lastNodePosition.y + y,
        this.lastNodePosition.z,
      );
      positions.needsUpdate = true;

      const farFade = 1 - smoothstep(FADE_START, FADE_END, visual.y);
      const verticalVisible = smoothstep(0.75, 1.7, visual.y);
      const targetOpacity = selected ? farFade * verticalVisible : 0;
      visual.opacity += (targetOpacity - visual.opacity) * opacityDamping;
      visual.reveal += (1 - visual.reveal) * revealDamping;
      visual.pulse *= Math.exp(-delta * 5.5);
      const revealScale = 0.93 + visual.reveal * 0.07;
      const pulseScale = 1 + visual.pulse * 0.028;
      visual.card.scale.setScalar(revealScale * pulseScale);
      visual.card.material.opacity = visual.opacity * visual.reveal * 0.9;
      visual.dot.material.opacity = visual.opacity;
      visual.line.material.opacity = visual.opacity * 0.3;
      visual.card.visible = visual.opacity > 0.012 && visual.reveal > 0.01;
      visual.dot.visible = visual.opacity > 0.012;
      visual.line.visible = visual.opacity > 0.012;
    }

    const composerTargetY = BASE_Y + this.scroll;
    this.composerY += (composerTargetY - this.composerY) * positionDamping;
    const composerTargetOpacity =
      selected && this.composerVisible
        ? (1 - smoothstep(FADE_START, FADE_END, this.composerY)) *
          smoothstep(0.75, 1.7, this.composerY)
        : 0;
    this.composerOpacity +=
      (composerTargetOpacity - this.composerOpacity) * opacityDamping;
    const composerAnchor = new THREE.Vector3(
      this.lastNodePosition.x,
      this.lastNodePosition.y + this.composerY,
      this.lastNodePosition.z,
    );
    const composerRight =
      cameraRight.lengthSq() > 0.0001
        ? cameraRight.clone().normalize()
        : new THREE.Vector3(1, 0, 0);
    this.composerPose.position
      .copy(composerAnchor)
      .addScaledVector(composerRight, CARD_CENTER_X);
    this.composerPose.rotationY = cardRotation;
    this.composerPose.opacity = this.composerOpacity;
    this.composerDot.position.copy(composerAnchor);
    this.composerDot.material.opacity = this.composerOpacity;
    this.composerDot.visible = this.composerOpacity > 0.012;
    const composerPositions = this.composerLine.geometry.getAttribute(
      "position",
    ) as THREE.BufferAttribute;
    composerPositions.setXYZ(
      0,
      this.lastNodePosition.x,
      this.lastNodePosition.y,
      this.lastNodePosition.z,
    );
    composerPositions.setXYZ(
      1,
      this.lastNodePosition.x,
      this.lastNodePosition.y + this.composerY,
      this.lastNodePosition.z,
    );
    composerPositions.needsUpdate = true;
    this.composerLine.material.opacity = this.composerOpacity * 0.3;
    this.composerLine.visible = this.composerOpacity > 0.012;

    const showEmpty =
      selected &&
      this.emptyEnabled &&
      !this.composerVisible &&
      !this.commentsLoading &&
      this.visuals.size === 0;
    this.emptyOpacity +=
      ((showEmpty ? 1 : 0) - this.emptyOpacity) * opacityDamping;
    this.emptyLabel.position.set(
      this.lastNodePosition.x,
      this.lastNodePosition.y + BASE_Y + 0.2,
      this.lastNodePosition.z,
    );
    this.emptyLabel.material.opacity = this.emptyOpacity;
    this.emptyLabel.visible = this.emptyOpacity > 0.012;
    this.group.visible =
      this.emptyOpacity > 0.012 ||
      this.composerOpacity > 0.012 ||
      [...this.visuals.values()].some((visual) => visual.opacity > 0.012);
  }

  public dispose() {
    for (const visual of this.visuals.values()) {
      this.disposeVisual(visual);
    }
    this.visuals.clear();
    this.cardGeometry.dispose();
    this.dotGeometry.dispose();
    this.composerDot.material.dispose();
    this.composerLine.geometry.dispose();
    this.composerLine.material.dispose();
    this.emptyLabel.material.map?.dispose();
    this.emptyLabel.material.dispose();
    this.group.clear();
  }

  private createVisual(item: ProgressItem): ProgressVisual {
    const texture = createProgressTexture(item, this.theme);
    const card = new THREE.Mesh(
      this.cardGeometry,
      new THREE.MeshBasicMaterial({
        map: texture,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        depthTest: false,
        side: THREE.DoubleSide,
      }),
    );
    card.userData.progressKey = progressKey(item);
    if (item.kind === "comment") {
      card.userData.commentId = item.comment.id;
    }
    const dot = new THREE.Mesh(
      this.dotGeometry,
      new THREE.MeshBasicMaterial({
        color:
          item.kind === "comment" && item.comment.reactions.length > 0
            ? 0x6d_e2_c2
            : 0xe7_b3_4d,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        depthTest: false,
        blending: THREE.AdditiveBlending,
      }),
    );
    const anchor = new THREE.Group();
    card.position.set(CARD_CENTER_X, 0, 0);
    dot.position.set(0, 0, 0);
    anchor.add(card, dot);
    const lineGeometry = new THREE.BufferGeometry().setFromPoints([
      new THREE.Vector3(0, 0, 0),
      new THREE.Vector3(0, 1, 0),
    ]);
    const line = new THREE.Line(
      lineGeometry,
      new THREE.LineBasicMaterial({
        color: 0xd8_a8_4c,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        depthTest: false,
        blending: THREE.NormalBlending,
      }),
    );

    return {
      anchor,
      card,
      dot,
      line,
      textureKey: progressTextureKey(item),
      y: BASE_Y,
      targetY: BASE_Y,
      opacity: 0,
      reveal: 0,
      pulse: 0,
      item,
    };
  }

  private disposeVisual(visual: ProgressVisual) {
    visual.card.material.map?.dispose();
    visual.card.material.dispose();
    visual.dot.material.dispose();
    visual.line.geometry.dispose();
    visual.line.material.dispose();
  }

  private progressY(count: number, index: number): number {
    const composerOffset = this.composerVisible ? 1 : 0;

    return BASE_Y + (count - 1 - index + composerOffset) * SPACING;
  }

  private minScroll(): number {
    if (this.visuals.size === 0) {
      return 0;
    }
    const composerOffset = this.composerVisible ? 1 : 0;

    return -(this.visuals.size - 1 + composerOffset) * SPACING;
  }

  private clampScroll() {
    const minimum = this.minScroll();
    this.targetScroll = Math.max(minimum, Math.min(0, this.targetScroll));
    this.scroll = Math.max(minimum, Math.min(0, this.scroll));
  }
}
