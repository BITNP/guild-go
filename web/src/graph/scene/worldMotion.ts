import * as THREE from "three";

const LAYOUT_ROTATION_SPEED = THREE.MathUtils.degToRad(0.45);
const VIRTUAL_ROOT_ROTATION_SPEED = (Math.PI * 2) / 80;
const VIRTUAL_ROOT_ROTATION_DELAY_MS = 800;
const ROTATION_SPEED_RESPONSE = 2.4;
const GRAVITY_ROLL_STRENGTH = 0.034;
const GRAVITY_PITCH_STRENGTH = 0.027;

interface WorldMotionInput {
  focusActive: boolean;
  pointerX: number;
  pointerY: number;
  selectedVirtual: boolean;
}

export class WorldMotion {
  private readonly world: THREE.Group;
  private readonly layoutRotation = new THREE.Quaternion();
  private readonly gravityTilt = new THREE.Quaternion();
  private readonly cameraForward = new THREE.Vector3();
  private readonly cameraRight = new THREE.Vector3();
  private layoutYaw = 0;
  private layoutRotationSpeed = LAYOUT_ROTATION_SPEED;
  private virtualFocusStartedAt = 0;

  constructor(world: THREE.Group) {
    this.world = world;
  }

  public get yaw() {
    return this.layoutYaw;
  }

  public startVirtualFocus(now: number) {
    this.virtualFocusStartedAt = now;
  }

  public update(
    delta: number,
    now: number,
    camera: THREE.Camera,
    input: WorldMotionInput,
  ) {
    const virtualRotationReady =
      input.selectedVirtual &&
      now - this.virtualFocusStartedAt >= VIRTUAL_ROOT_ROTATION_DELAY_MS;
    const targetRotationSpeed = input.focusActive
      ? virtualRotationReady
        ? VIRTUAL_ROOT_ROTATION_SPEED
        : 0
      : LAYOUT_ROTATION_SPEED;
    const rotationDamping = 1 - Math.exp(-delta * ROTATION_SPEED_RESPONSE);

    this.layoutRotationSpeed +=
      (targetRotationSpeed - this.layoutRotationSpeed) * rotationDamping;
    this.layoutYaw += delta * this.layoutRotationSpeed;
    this.layoutRotation.setFromAxisAngle(
      new THREE.Vector3(0, 1, 0),
      this.layoutYaw,
    );

    this.cameraForward
      .set(0, 0, -1)
      .applyQuaternion(camera.quaternion)
      .normalize();
    this.cameraRight
      .set(1, 0, 0)
      .applyQuaternion(camera.quaternion)
      .normalize();

    const pointerInfluence = input.focusActive ? 0.5 : 1;
    this.gravityTilt.setFromAxisAngle(
      this.cameraForward,
      input.pointerX * GRAVITY_ROLL_STRENGTH * pointerInfluence,
    );
    this.gravityTilt.multiply(
      new THREE.Quaternion().setFromAxisAngle(
        this.cameraRight,
        input.pointerY * GRAVITY_PITCH_STRENGTH * pointerInfluence,
      ),
    );

    this.world.quaternion
      .copy(this.layoutRotation)
      .premultiply(this.gravityTilt);
  }
}
