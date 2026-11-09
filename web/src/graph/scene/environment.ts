import * as THREE from "three";

import type { ThemeMode } from "@/composables/useTheme";

import type { GraphGround } from "../ground";
import { createGround } from "../ground";
import { createDust } from "../visuals";
import { SCENE_PALETTES } from "./palette";

export class GraphEnvironment {
  private readonly scene: THREE.Scene;
  private readonly world: THREE.Group;
  private readonly ground: GraphGround;
  private readonly dust: THREE.Points<
    THREE.BufferGeometry,
    THREE.PointsMaterial
  >;

  private readonly lights: THREE.Light[];
  private theme: ThemeMode;

  constructor(scene: THREE.Scene, world: THREE.Group, theme: ThemeMode) {
    this.scene = scene;
    this.world = world;
    this.theme = theme;

    this.scene.background = new THREE.Color(this.palette.background);
    this.scene.fog = new THREE.FogExp2(this.palette.fog, 0.0024);

    this.ground = createGround(this.palette);
    this.world.add(this.ground.group);

    this.dust = createDust();
    this.scene.add(this.dust);

    const hemisphere = new THREE.HemisphereLight(0xff_e2_a7, 0x07_10_18, 2.2);
    const key = new THREE.DirectionalLight(0xff_e0_a0, 2.7);
    key.position.set(28, 60, 34);

    const rim = new THREE.DirectionalLight(0x77_b9_d1, 1.2);
    rim.position.set(-36, 20, -28);

    this.lights = [hemisphere, key, rim];
    this.scene.add(...this.lights);
  }

  private get palette() {
    return SCENE_PALETTES[this.theme];
  }

  public setTheme(theme: ThemeMode) {
    this.theme = theme;
    this.scene.background = new THREE.Color(this.palette.background);

    if (this.scene.fog instanceof THREE.FogExp2) {
      this.scene.fog.color.setHex(this.palette.fog);
    }

    this.ground.plane.material.color.setHex(this.palette.ground);
    this.ground.grid.material.uniforms.minorColor?.value.setHex(
      this.palette.minor,
    );
    this.ground.grid.material.uniforms.accentColor?.value.setHex(
      this.palette.accent,
    );
    this.dust.material.color.setHex(this.palette.dust);
    this.dust.material.opacity = this.palette.dustOpacity;
  }

  public update(delta: number) {
    this.dust.rotation.y += delta * 0.004;
  }

  public dispose() {
    this.world.remove(this.ground.group);
    this.scene.remove(this.dust, ...this.lights);

    this.ground.group.traverse((object) => {
      if (!(object instanceof THREE.Mesh)) {
        return;
      }
      object.geometry.dispose();
      object.material.dispose();
    });

    this.dust.geometry.dispose();
    this.dust.material.dispose();
  }
}
