import * as THREE from "three";

import { nearestAngle } from "../math";
import { PROGRESS_CARD_WIDTH } from "../progressLayout";

const INITIAL_PITCH = THREE.MathUtils.degToRad(60);
const INITIAL_YAW = THREE.MathUtils.degToRad(24);
const INTRO_DURATION_MS = 1700;
const INTRO_START_DISTANCE_MULTIPLIER = 3.2;

interface CameraSnapshot {
  distance: number;
  pitch: number;
  target: THREE.Vector3;
}

export class CameraRig {
  public readonly camera = new THREE.PerspectiveCamera(44, 1, 0.1, 1800);

  private readonly viewTarget = new THREE.Vector3();
  private readonly desiredTarget = new THREE.Vector3();
  private readonly dragRight = new THREE.Vector3();
  private readonly dragForward = new THREE.Vector3();
  private readonly focusNodePosition = new THREE.Vector3();

  private focusNodeVisible = false;
  private focusNodeVirtual = false;
  private snapshot: CameraSnapshot | null = null;
  private distance = 90;
  private desiredDistance = 90;
  private fitDistance = 90;
  private pitch = INITIAL_PITCH;
  private desiredPitch = this.pitch;
  private yaw = INITIAL_YAW;
  private desiredYaw = this.yaw;
  private introStarted = false;
  private introActive = false;
  private introStartedAt = 0;
  private introFromDistance = 90;
  private introToDistance = 90;
  private introFromPitch = 0;
  private introToPitch = INITIAL_PITCH;
  private introFromYaw = INITIAL_YAW;
  private introToYaw = INITIAL_YAW;

  constructor() {
    this.camera.position.set(0, 88, 20);
    this.camera.lookAt(0, 0, 0);
  }

  public get currentDistance() {
    return this.distance;
  }

  public get fittedDistance() {
    return this.fitDistance;
  }

  public get targetYaw() {
    return this.desiredYaw;
  }

  public resize(width: number, height: number) {
    this.camera.aspect = width / height;
    this.camera.updateProjectionMatrix();

    if (this.focusNodeVisible) {
      if (!this.focusNodeVirtual) {
        this.desiredDistance = this.nodeFocusDistance();
      }
      this.desiredTarget.copy(
        this.focusTarget(
          this.focusNodePosition,
          this.desiredYaw,
          this.desiredPitch,
          this.desiredDistance,
          this.focusNodeVirtual ? 0 : 0.18,
        ),
      );
    }
  }

  public fitView(sceneRadius: number) {
    this.desiredTarget.set(0, 0.6, 0);
    this.viewTarget.copy(this.desiredTarget);
    this.desiredDistance = this.overviewFitDistance(sceneRadius);
    this.fitDistance = this.desiredDistance;
    this.desiredPitch = INITIAL_PITCH;
    this.desiredYaw = nearestAngle(this.yaw, INITIAL_YAW);
    this.focusNodeVisible = false;
    this.focusNodeVirtual = false;
    this.snapshot = {
      target: this.desiredTarget.clone(),
      distance: this.desiredDistance,
      pitch: this.desiredPitch,
    };

    if (!this.introStarted) {
      this.startIntro();

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

  public focus(
    nodePosition: THREE.Vector3,
    virtual: boolean,
    sceneRadius: number,
    captureSnapshot: boolean,
  ) {
    this.introActive = false;
    this.focusNodePosition.copy(nodePosition);
    this.focusNodeVisible = true;
    this.focusNodeVirtual = virtual;

    if (captureSnapshot && !this.snapshot) {
      this.snapshot = {
        target: this.viewTarget.clone(),
        distance: this.distance,
        pitch: this.pitch,
      };
    }

    if (virtual) {
      this.desiredDistance = this.overviewFitDistance(sceneRadius, 1.12);
      this.fitDistance = this.desiredDistance;
      this.desiredPitch = Math.PI / 2 - 0.018;
      this.desiredYaw = this.yaw;
      this.desiredTarget.copy(
        this.focusTarget(
          new THREE.Vector3(0, 0.9, 0),
          this.desiredYaw,
          this.desiredPitch,
          this.desiredDistance,
          0,
        ),
      );

      return;
    }

    const radialYaw =
      this.focusNodePosition.lengthSq() > 0.01
        ? Math.atan2(this.focusNodePosition.x, this.focusNodePosition.z)
        : this.yaw;

    this.desiredDistance = this.nodeFocusDistance();
    this.desiredPitch = THREE.MathUtils.degToRad(21);
    this.desiredYaw = nearestAngle(this.yaw, radialYaw);
    this.desiredTarget.copy(
      this.focusTarget(
        this.focusNodePosition,
        this.desiredYaw,
        this.desiredPitch,
        this.desiredDistance,
      ),
    );
  }

  public clearFocus() {
    this.focusNodeVisible = false;
    this.focusNodeVirtual = false;

    if (!this.snapshot) {
      return;
    }

    this.desiredTarget.copy(this.snapshot.target);
    this.desiredDistance = this.snapshot.distance;
    this.desiredPitch = this.snapshot.pitch;
    this.desiredYaw = this.yaw;
  }

  public stopIntro() {
    this.introActive = false;
  }

  public zoom(deltaY: number) {
    this.introActive = false;
    this.desiredDistance = THREE.MathUtils.clamp(
      this.desiredDistance * Math.exp(deltaY * 0.0011),
      16,
      440,
    );
  }

  public panByPixels(dx: number, dy: number, viewportHeight: number) {
    const scale = (this.distance / Math.max(viewportHeight, 1)) * 1.15;

    this.dragRight.set(1, 0, 0).applyQuaternion(this.camera.quaternion).setY(0);
    if (this.dragRight.lengthSq() < 0.0001) {
      this.dragRight.set(Math.cos(this.yaw), 0, -Math.sin(this.yaw));
    } else {
      this.dragRight.normalize();
    }

    this.dragForward
      .set(0, 0, -1)
      .applyQuaternion(this.camera.quaternion)
      .setY(0);
    if (this.dragForward.lengthSq() < 0.0001) {
      this.dragForward.set(-Math.sin(this.yaw), 0, -Math.cos(this.yaw));
    } else {
      this.dragForward.normalize();
    }

    this.desiredTarget
      .addScaledVector(this.dragRight, -dx * scale)
      .addScaledVector(this.dragForward, dy * scale);
  }

  public update(delta: number, now: number) {
    const damping = 1 - Math.exp(-delta * 5.2);

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

    this.camera.position.copy(this.viewTarget).add(cameraOffset);
    this.camera.lookAt(this.viewTarget);
    this.camera.updateMatrixWorld();
    this.clearSettledSnapshot();
  }

  private startIntro() {
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
  }

  private clearSettledSnapshot() {
    if (this.focusNodeVisible || this.introActive || !this.snapshot) {
      return;
    }

    const settled =
      this.viewTarget.distanceToSquared(this.snapshot.target) < 0.0004 &&
      Math.abs(this.distance - this.snapshot.distance) < 0.05 &&
      Math.abs(this.pitch - this.snapshot.pitch) < 0.001;

    if (settled) {
      this.snapshot = null;
    }
  }

  private overviewFitDistance(sceneRadius: number, padding = 1.08): number {
    const verticalFov = THREE.MathUtils.degToRad(this.camera.fov);
    const horizontalFov =
      2 * Math.atan(Math.tan(verticalFov / 2) * this.camera.aspect);
    const radius = Math.max(sceneRadius + 6, 16);
    const horizontalDistance = radius / Math.tan(horizontalFov / 2);
    const verticalDistance = radius / Math.tan(verticalFov / 2);

    return Math.max(28, horizontalDistance, verticalDistance) * padding;
  }

  private nodeFocusDistance(): number {
    const verticalFov = THREE.MathUtils.degToRad(this.camera.fov);
    const fitWidth = PROGRESS_CARD_WIDTH + 2;
    const horizontalDistance =
      fitWidth / (2 * Math.tan(verticalFov / 2) * this.camera.aspect);

    return Math.max(
      THREE.MathUtils.clamp(this.camera.fov / 2.2, 16, 21),
      horizontalDistance,
    );
  }

  private focusTarget(
    nodePosition: THREE.Vector3,
    yaw: number,
    pitch: number,
    distance: number,
    verticalOffset = 0.18,
  ): THREE.Vector3 {
    const verticalExtent =
      2 * distance * Math.tan(THREE.MathUtils.degToRad(this.camera.fov / 2));
    const horizontalExtent = verticalExtent * this.camera.aspect;
    const horizontalOffset = this.focusNodeVirtual
      ? horizontalExtent * 0.16
      : Math.max(horizontalExtent * 0.16, PROGRESS_CARD_WIDTH / 2);
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
      .addScaledVector(right, horizontalOffset)
      .addScaledVector(screenUp, verticalExtent * verticalOffset);
  }
}
