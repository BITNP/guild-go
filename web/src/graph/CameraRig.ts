import * as THREE from "three";

const INITIAL_PITCH = THREE.MathUtils.degToRad(60);
const INITIAL_YAW = THREE.MathUtils.degToRad(24);
const INTRO_DURATION_MS = 1700;
const INTRO_START_DISTANCE_MULTIPLIER = 3.2;

export class CameraRig {
  public hasInteracted = false;

  private readonly viewTarget = new THREE.Vector3();
  private readonly desiredTarget = new THREE.Vector3();
  private readonly dragRight = new THREE.Vector3();
  private readonly dragForward = new THREE.Vector3();
  private snapshot: {
    target: THREE.Vector3;
    distance: number;
    pitch: number;
  } | null = null;

  private distance = 90;
  private desiredDistance = 90;
  private fitDistance = 90;
  private pitch = INITIAL_PITCH;
  private desiredPitch = INITIAL_PITCH;
  private yaw = INITIAL_YAW;
  private desiredYaw = INITIAL_YAW;
  private focusNodePosition: THREE.Vector3 | null = null;
  private introStarted = false;
  private introActive = false;
  private introStartedAt = 0;
  private introFromDistance = 90;
  private introToDistance = 90;
  private introFromPitch = 0;
  private introToPitch = INITIAL_PITCH;
  private introFromYaw = INITIAL_YAW;
  private introToYaw = INITIAL_YAW;

  constructor(camera: THREE.PerspectiveCamera) {
    camera.position.set(0, 88, 20);
    camera.lookAt(0, 0, 0);
  }

  public getDesiredYaw(): number {
    return this.desiredYaw;
  }

  public zoomRatio(): number {
    return this.fitDistance / Math.max(this.distance, 1);
  }

  public cancelIntro() {
    this.introActive = false;
  }

  public saveSnapshot() {
    if (this.snapshot) {
      return;
    }
    this.snapshot = {
      target: this.viewTarget.clone(),
      distance: this.distance,
      pitch: this.pitch,
    };
  }

  public fitOverview(camera: THREE.PerspectiveCamera, layoutRadius: number) {
    this.desiredTarget.set(0, 0.6, 0);
    this.viewTarget.copy(this.desiredTarget);
    this.desiredDistance = this.overviewFitDistance(camera, layoutRadius);
    this.fitDistance = this.desiredDistance;
    this.desiredPitch = INITIAL_PITCH;
    this.desiredYaw = nearestAngle(this.yaw, INITIAL_YAW);
    this.focusNodePosition = null;
    this.snapshot = {
      target: this.desiredTarget.clone(),
      distance: this.desiredDistance,
      pitch: this.desiredPitch,
    };
    if (!this.introStarted) {
      this.introStarted = true;
      this.introActive = true;
      this.introStartedAt = performance.now();
      this.distance = this.desiredDistance * INTRO_START_DISTANCE_MULTIPLIER;
      this.pitch = 0;
      this.yaw = this.desiredYaw;
      this.introFromDistance = this.distance;
      this.introToDistance = this.desiredDistance;
      this.introFromPitch = 0;
      this.introToPitch = this.desiredPitch;
      this.introFromYaw = this.yaw;
      this.introToYaw = this.desiredYaw;

      return;
    }
    if (this.introActive) {
      this.introToDistance = this.desiredDistance;
      this.introToPitch = this.desiredPitch;
      this.introToYaw = this.desiredYaw;

      return;
    }
    this.distance = this.desiredDistance;
    this.pitch = this.desiredPitch;
    this.yaw = this.desiredYaw;
  }

  public focusVirtualRoot(
    camera: THREE.PerspectiveCamera,
    layoutRadius: number,
    worldPosition: THREE.Vector3,
  ) {
    this.introActive = false;
    this.focusNodePosition = worldPosition.clone();
    this.desiredDistance = this.overviewFitDistance(camera, layoutRadius, 1.12);
    this.fitDistance = this.desiredDistance;
    this.desiredPitch = Math.PI / 2 - 0.018;
    this.desiredYaw = this.yaw;
    this.desiredTarget.copy(
      this.focusTarget(
        camera,
        new THREE.Vector3(0, 0.9, 0),
        this.desiredYaw,
        this.desiredPitch,
        this.desiredDistance,
        0,
      ),
    );
    this.hasInteracted = true;
  }

  public focusNode(
    camera: THREE.PerspectiveCamera,
    worldPosition: THREE.Vector3,
  ) {
    this.introActive = false;
    this.focusNodePosition = worldPosition.clone();
    const radialYaw =
      worldPosition.lengthSq() > 0.01
        ? Math.atan2(worldPosition.x, worldPosition.z)
        : this.yaw;
    this.desiredDistance = Math.max(16, Math.min(21, camera.fov / 2.2));
    this.desiredPitch = THREE.MathUtils.degToRad(21);
    this.desiredYaw = nearestAngle(this.yaw, radialYaw);
    this.desiredTarget.copy(
      this.focusTarget(
        camera,
        this.focusNodePosition,
        this.desiredYaw,
        this.desiredPitch,
        this.desiredDistance,
      ),
    );
    this.hasInteracted = true;
  }

  public clearFocus() {
    this.focusNodePosition = null;
    this.hasInteracted = true;
    if (!this.snapshot) {
      return;
    }
    this.desiredTarget.copy(this.snapshot.target);
    this.desiredDistance = this.snapshot.distance;
    this.desiredPitch = this.snapshot.pitch;
    // Keep the frozen layout bearing and return only the overview framing.
    this.desiredYaw = this.yaw;
  }

  public pan(
    camera: THREE.PerspectiveCamera,
    dx: number,
    dy: number,
    viewportHeight: number,
  ) {
    this.hasInteracted = true;
    const scale = (this.distance / Math.max(viewportHeight, 1)) * 1.15;
    this.dragRight.set(1, 0, 0).applyQuaternion(camera.quaternion);
    this.dragRight.y = 0;
    if (this.dragRight.lengthSq() < 0.0001) {
      this.dragRight.set(Math.cos(this.yaw), 0, -Math.sin(this.yaw));
    } else {
      this.dragRight.normalize();
    }
    this.dragForward.set(0, 0, -1).applyQuaternion(camera.quaternion);
    this.dragForward.y = 0;
    if (this.dragForward.lengthSq() < 0.0001) {
      this.dragForward.set(-Math.sin(this.yaw), 0, -Math.cos(this.yaw));
    } else {
      this.dragForward.normalize();
    }
    this.desiredTarget
      .addScaledVector(this.dragRight, -dx * scale)
      .addScaledVector(this.dragForward, dy * scale);
  }

  public zoom(deltaY: number) {
    this.introActive = false;
    this.hasInteracted = true;
    this.desiredDistance = THREE.MathUtils.clamp(
      this.desiredDistance * Math.exp(deltaY * 0.0011),
      16,
      440,
    );
  }

  public onResize(
    camera: THREE.PerspectiveCamera,
    isFocused: boolean,
    isVirtualRoot: boolean,
  ) {
    if (!isFocused || !this.focusNodePosition) {
      return;
    }
    this.desiredTarget.copy(
      this.focusTarget(
        camera,
        this.focusNodePosition,
        this.desiredYaw,
        this.desiredPitch,
        this.desiredDistance,
        isVirtualRoot ? 0 : 0.18,
      ),
    );
  }

  public update(
    camera: THREE.PerspectiveCamera,
    now: number,
    damping: number,
    isFocused: boolean,
  ) {
    this.viewTarget.lerp(this.desiredTarget, damping);
    if (this.introActive) {
      const elapsed = now - this.introStartedAt;
      const progress = THREE.MathUtils.clamp(elapsed / INTRO_DURATION_MS, 0, 1);
      const eased = 1 - (1 - progress) ** 3;
      this.distance = THREE.MathUtils.lerp(
        this.introFromDistance,
        this.introToDistance,
        eased,
      );
      this.pitch = THREE.MathUtils.lerp(
        this.introFromPitch,
        this.introToPitch,
        eased,
      );
      this.yaw = THREE.MathUtils.lerp(
        this.introFromYaw,
        this.introToYaw,
        eased,
      );
      if (progress >= 1) {
        this.introActive = false;
      }
    } else {
      this.distance += (this.desiredDistance - this.distance) * damping;
      this.pitch += (this.desiredPitch - this.pitch) * damping;
      this.yaw += (this.desiredYaw - this.yaw) * damping;
    }
    const horizontalDistance = Math.cos(this.pitch) * this.distance;
    const cameraOffset = new THREE.Vector3(
      Math.sin(this.yaw) * horizontalDistance,
      Math.sin(this.pitch) * this.distance,
      Math.cos(this.yaw) * horizontalDistance,
    );
    camera.position.copy(this.viewTarget).add(cameraOffset);
    camera.lookAt(this.viewTarget);
    camera.updateMatrixWorld();
    if (!isFocused && !this.introActive && this.snapshot) {
      const snapshotSettled =
        this.viewTarget.distanceToSquared(this.snapshot.target) < 0.0004 &&
        Math.abs(this.distance - this.snapshot.distance) < 0.05 &&
        Math.abs(this.pitch - this.snapshot.pitch) < 0.001;
      if (snapshotSettled) {
        this.snapshot = null;
      }
    }
  }

  private overviewFitDistance(
    camera: THREE.PerspectiveCamera,
    layoutRadius: number,
    padding = 1.08,
  ): number {
    const verticalFov = THREE.MathUtils.degToRad(camera.fov);
    const horizontalFov =
      2 * Math.atan(Math.tan(verticalFov / 2) * camera.aspect);
    const sceneRadius = Math.max(layoutRadius + 6, 16);
    const horizontalDistance = sceneRadius / Math.tan(horizontalFov / 2);
    const verticalDistance = sceneRadius / Math.tan(verticalFov / 2);

    return Math.max(28, horizontalDistance, verticalDistance) * padding;
  }

  private focusTarget(
    camera: THREE.PerspectiveCamera,
    nodePosition: THREE.Vector3,
    yaw: number,
    pitch: number,
    distance: number,
    verticalOffset = 0.18,
  ): THREE.Vector3 {
    const verticalExtent =
      2 * distance * Math.tan(THREE.MathUtils.degToRad(camera.fov / 2));
    const horizontalExtent = verticalExtent * camera.aspect;
    const offsetDirection = new THREE.Vector3(
      Math.sin(yaw) * Math.cos(pitch),
      Math.sin(pitch),
      Math.cos(yaw) * Math.cos(pitch),
    );
    const forward = offsetDirection.multiplyScalar(-1).normalize();
    const right = new THREE.Vector3()
      .crossVectors(forward, new THREE.Vector3(0, 1, 0))
      .normalize();
    const screenUp = new THREE.Vector3()
      .crossVectors(right, forward)
      .normalize();

    return nodePosition
      .clone()
      .addScaledVector(right, horizontalExtent * 0.16)
      .addScaledVector(screenUp, verticalExtent * verticalOffset);
  }
}

function nearestAngle(current: number, target: number): number {
  const turn = Math.PI * 2;

  return (
    current +
    THREE.MathUtils.euclideanModulo(target - current + Math.PI, turn) -
    Math.PI
  );
}
