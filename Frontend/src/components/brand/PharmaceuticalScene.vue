<script setup>
defineProps({ compact: { type: Boolean, default: false } })
const products = [
  { name: 'Panadol', line: 'HEALTH & CARE', tone: 'red', position: 'one' },
  { name: 'Promag', line: 'DAILY WELLNESS', tone: 'green', position: 'two' },
  { name: 'bodrex', line: 'PHARMACEUTICAL', tone: 'orange', position: 'three' },
]
</script>
<template>
  <div
    class="pharma-scene"
    :class="{ compact }"
    aria-label="Ilustrasi tiga dimensi kemasan produk farmasi"
  >
    <div class="scene-orbit orbit-one"></div>
    <div class="scene-orbit orbit-two"></div>
    <div class="scene-grid"></div>
    <div class="scene-platform"><span></span></div>
    <div
      v-for="product in products"
      :key="product.name"
      class="medicine-float"
      :class="product.position"
    >
      <div class="medicine-box" :class="product.tone">
        <div class="box-front">
          <span class="pack-cross">+</span><small>{{ product.line }}</small
          ><strong>{{ product.name }}</strong
          ><span class="pack-rule"></span
          ><span class="pack-bottom">ILUSTRASI PRODUK</span>
        </div>
        <div class="box-side">
          <span>{{ product.name }}</span>
        </div>
        <div class="box-top"></div>
      </div>
    </div>
    <div class="vitamin-bottle">
      <div class="bottle-cap"></div>
      <div class="bottle-body">
        <div class="bottle-label">
          <span>FAS</span><strong>Vitamin<br />Collection</strong
          ><small>HEALTH ESSENTIALS</small>
        </div>
      </div>
    </div>
    <div class="floating-pill pill-a"></div>
    <div class="floating-pill pill-b"></div>
    <div class="floating-pill pill-c"></div>
    <div class="scene-badge">
      <span class="scene-badge-icon">✦</span>
      <div>
        <strong>Care in every connection.</strong
        ><small>PT. FAS · Business Platform</small>
      </div>
    </div>
    <span class="scene-caption">Ilustrasi visual kemasan</span>
  </div>
</template>
<style scoped>
.pharma-scene {
  position: relative;
  width: 100%;
  height: 510px;
  isolation: isolate;
  perspective: 1100px;
  overflow: hidden;
  --box-depth: 26px;
}
.scene-grid {
  position: absolute;
  inset: 30% -10% -10%;
  background-image:
    linear-gradient(var(--fas-accent-border) 1px, transparent 1px),
    linear-gradient(90deg, var(--fas-accent-border) 1px, transparent 1px);
  background-size: 42px 42px;
  transform: rotateX(65deg);
  opacity: 0.22;
  mask-image: radial-gradient(ellipse, black, transparent 70%);
}
.scene-orbit {
  position: absolute;
  left: 8%;
  top: 8%;
  width: 84%;
  height: 78%;
  border: 1px solid var(--fas-accent-border);
  border-radius: 50%;
  transform: rotate(-25deg);
  opacity: 0.65;
}
.orbit-two {
  inset: 17% 5% auto;
  width: 90%;
  height: 58%;
  transform: rotate(26deg);
  opacity: 0.35;
}
.scene-platform {
  position: absolute;
  width: 77%;
  height: 105px;
  border-radius: 50%;
  left: 11%;
  bottom: 50px;
  background: linear-gradient(
    160deg,
    color-mix(in srgb, var(--fas-accent) 16%, white),
    color-mix(in srgb, var(--fas-accent) 38%, #dbe9ea)
  );
  box-shadow:
    0 16px 0 color-mix(in srgb, var(--fas-accent) 30%, #a3bec3),
    0 35px 45px -12px color-mix(in srgb, var(--fas-accent) 30%, transparent);
  transform: rotate(-6deg);
}
.scene-platform span {
  position: absolute;
  inset: 10px;
  border: 1px solid #ffffff80;
  border-radius: 50%;
}
.medicine-float {
  position: absolute;
  transform-style: preserve-3d;
  animation: medicine-drift 7s ease-in-out infinite;
}
.one {
  left: 15%;
  top: 18%;
  z-index: 3;
  transform: rotate(-13deg);
}
.two {
  left: 47%;
  top: 8%;
  z-index: 2;
  animation-delay: -2.3s;
  transform: rotate(12deg);
}
.three {
  left: 58%;
  top: 48%;
  z-index: 5;
  animation-delay: -4.1s;
  transform: rotate(-8deg) scale(0.85);
}
.medicine-box {
  width: 142px;
  height: 200px;
  position: relative;
  transform-style: preserve-3d;
  transform: rotateY(-25deg) rotateX(8deg);
  --pack: #d52943;
  --pack-side: #9f1732;
  filter: drop-shadow(16px 25px 15px #102e4225);
}
.medicine-box.green {
  --pack: #178e6d;
  --pack-side: #086a4e;
  height: 210px;
}
.medicine-box.orange {
  --pack: #e65d32;
  --pack-side: #ab3c20;
  width: 165px;
  height: 120px;
}
.box-front {
  position: absolute;
  inset: 0;
  background: linear-gradient(120deg, #fff 15%, #f1f4f4 100%);
  border-radius: 5px 0 0 5px;
  overflow: hidden;
  border: 1px solid #d9e1e1;
  display: flex;
  flex-direction: column;
  padding: 17px 13px;
  color: #12333c;
  backface-visibility: hidden;
}
.box-front:before {
  content: '';
  position: absolute;
  height: 44%;
  inset: 0 0 auto;
  background: linear-gradient(130deg, var(--pack), var(--pack-side));
  z-index: 0;
}
.box-front > * {
  position: relative;
  z-index: 1;
}
.pack-cross {
  position: absolute;
  right: 11px;
  top: 7px;
  color: #ffffffc9;
  font-size: 21px;
}
.box-front > small {
  font:
    7px/1.4 Arial,
    sans-serif;
  letter-spacing: 1px;
  color: #fff;
  margin-top: 5px;
}
.box-front > strong {
  font:
    bold italic 23px/1.2 Arial,
    sans-serif;
  margin-top: 14px;
  color: #fff;
  letter-spacing: -1px;
  white-space: nowrap;
}
.pack-rule {
  height: 7px;
  width: 70%;
  background: var(--pack);
  border-radius: 10px;
  margin-top: auto;
  box-shadow:
    0 12px 0 #d8e4e5,
    0 24px 0 #e5edec;
}
.pack-bottom {
  font:
    6px Arial,
    sans-serif;
  color: #718589;
  margin-top: 35px;
  letter-spacing: 1px;
}
.box-side {
  position: absolute;
  left: 100%;
  top: 0;
  width: var(--box-depth);
  height: 100%;
  background: linear-gradient(90deg, var(--pack-side), var(--pack));
  transform: rotateY(90deg);
  transform-origin: left;
  display: flex;
  align-items: center;
  justify-content: center;
  backface-visibility: hidden;
}
.box-side span {
  writing-mode: vertical-rl;
  color: #fff;
  font: bold 11px Arial;
  letter-spacing: 0.1em;
}
.box-top {
  position: absolute;
  bottom: 100%;
  left: 0;
  width: 100%;
  height: var(--box-depth);
  background: #edf3f3;
  transform: rotateX(90deg);
  transform-origin: bottom;
  border: 1px solid #d4e0e0;
}
.orange .box-front {
  padding: 10px;
}
.orange .box-front:before {
  height: 60%;
}
.orange .box-front > strong {
  font-size: 26px;
  margin-top: 6px;
}
.orange .pack-rule {
  margin-top: 16px;
  height: 4px;
}
.orange .pack-bottom {
  margin-top: 14px;
}
.vitamin-bottle {
  position: absolute;
  z-index: 4;
  left: 28%;
  bottom: 77px;
  width: 104px;
  height: 164px;
  transform: rotate(-12deg);
  filter: drop-shadow(15px 18px 12px #16324925);
  animation: bottle-drift 8s ease-in-out infinite;
}
.bottle-cap {
  position: absolute;
  left: 8px;
  top: 0;
  width: 88px;
  height: 29px;
  border-radius: 8px 8px 3px 3px;
  background: repeating-linear-gradient(90deg, #fafdfd 0 3px, #d8e4e6 3px 5px);
  box-shadow: inset 0 -5px 8px #9fb5be66;
  z-index: 2;
}
.bottle-body {
  position: absolute;
  inset: 20px 0 0;
  border-radius: 23px 23px 17px 17px;
  background: linear-gradient(
    90deg,
    #bb6324,
    #f3b765 25%,
    #d18a39 65%,
    #9c511b
  );
  border: 1px solid #c38b4c;
  overflow: hidden;
}
.bottle-body:after {
  content: '';
  position: absolute;
  left: 8px;
  top: 0;
  width: 12px;
  height: 100%;
  background: #fff4;
  filter: blur(3px);
}
.bottle-label {
  position: absolute;
  inset: 35px 0 16px;
  background: #f9faf5;
  padding: 8px 10px;
  color: #225856;
}
.bottle-label > span {
  font: bold 8px Arial;
  letter-spacing: 3px;
}
.bottle-label strong {
  display: block;
  font: bold 13px/1.2 Arial;
  margin-top: 4px;
}
.bottle-label small {
  font: 5px Arial;
  letter-spacing: 1px;
}
.floating-pill {
  position: absolute;
  width: 48px;
  height: 19px;
  border-radius: 20px;
  background: linear-gradient(90deg, #f8fcfc 50%, var(--fas-accent) 50%);
  box-shadow:
    inset 0 -5px 8px #17203328,
    0 9px 15px #17203315;
  animation: medicine-drift 6s ease-in-out infinite;
}
.pill-a {
  left: 8%;
  top: 48%;
  rotate: 35deg;
}
.pill-b {
  right: 12%;
  top: 30%;
  rotate: -35deg;
  animation-delay: -2s;
}
.pill-c {
  right: 25%;
  bottom: 70px;
  rotate: 20deg;
  width: 34px;
  height: 14px;
  animation-delay: -4s;
}
.scene-badge {
  position: absolute;
  z-index: 6;
  left: 4%;
  bottom: 20px;
  display: flex;
  align-items: center;
  gap: 11px;
  background: color-mix(in srgb, var(--fas-card) 88%, transparent);
  backdrop-filter: blur(16px);
  padding: 13px 17px;
  border: 1px solid var(--fas-border-strong);
  border-radius: 16px;
  box-shadow: var(--fas-shadow-soft);
  transform: rotate(-3deg);
}
.scene-badge-icon {
  font-size: 28px;
  color: var(--fas-accent-text);
}
.scene-badge strong {
  font-size: 11px;
  color: var(--fas-text);
}
.scene-badge small {
  display: block;
  font-size: 9px;
  color: var(--fas-muted);
  margin-top: 3px;
}
.scene-caption {
  position: absolute;
  right: 5%;
  bottom: 8px;
  font-size: 9px;
  color: var(--fas-muted);
}
.compact {
  height: 330px;
}
.compact .one {
  left: 21%;
  top: 15%;
  scale: 0.75;
}
.compact .two {
  left: 51%;
  top: 1%;
  scale: 0.7;
}
.compact .three {
  left: 62%;
  top: 47%;
  scale: 0.7;
}
.compact .vitamin-bottle {
  left: 32%;
  bottom: 55px;
  scale: 0.75;
}
.compact .scene-platform {
  bottom: 43px;
  height: 70px;
}
.compact .scene-badge {
  bottom: 12px;
  left: 5%;
  scale: 0.87;
}
.compact .scene-caption {
  bottom: 0;
}
@keyframes medicine-drift {
  0%,
  100% {
    translate: 0 0;
  }
  50% {
    translate: 0 -14px;
  }
}
@keyframes bottle-drift {
  0%,
  100% {
    translate: 0 0;
    rotate: 0deg;
  }
  50% {
    translate: 0 -9px;
    rotate: 4deg;
  }
}
@media (max-width: 600px) {
  .pharma-scene {
    height: 360px;
    --box-depth: 23px;
  }
  .one {
    left: 10%;
    top: 16%;
    scale: 0.75;
  }
  .two {
    left: 44%;
    top: 4%;
    scale: 0.75;
  }
  .three {
    left: 54%;
    top: 44%;
    scale: 0.65;
  }
  .vitamin-bottle {
    left: 24%;
    bottom: 45px;
    scale: 0.8;
  }
  .scene-platform {
    bottom: 35px;
    height: 80px;
  }
  .scene-badge {
    scale: 0.8;
    left: 0;
    bottom: 4px;
    transform-origin: left;
  }
  .scene-caption {
    bottom: 0;
    font-size: 8px;
  }
  .compact {
    height: 280px;
  }
  .compact .one {
    left: 15%;
    top: 6%;
    scale: 0.65;
  }
  .compact .two {
    left: 48%;
    top: -3%;
    scale: 0.6;
  }
  .compact .three {
    left: 53%;
    top: 38%;
    scale: 0.55;
  }
  .compact .vitamin-bottle {
    left: 26%;
    bottom: 32px;
    scale: 0.6;
  }
  .compact .scene-platform {
    bottom: 28px;
  }
  .compact .scene-badge {
    display: none;
  }
}
@media (prefers-reduced-motion: reduce) {
  .medicine-float,
  .vitamin-bottle,
  .floating-pill {
    animation: none;
  }
}
</style>
