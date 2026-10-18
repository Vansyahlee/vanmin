<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from "vue";
import { RouterLink } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import PharmaceuticalScene from "@/components/brand/PharmaceuticalScene.vue";
import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import Lenis from "lenis";
import "lenis/dist/lenis.css";

// GSAP = elemen berpencar -> tersusun. Lenis = gerakan scroll dengan inersia.
// Browser tetap bisa scroll biasa, tidak ada event.preventDefault() buatan.
gsap.registerPlugin(ScrollTrigger);

const auth = useAuthStore();
const landingRoot = ref(null);
const stage = ref(null);
const activeFeature = ref(0);
const turns = ref(0);
const pressed = ref(false);
const reducedMotion = ref(false);

function initialMode() {
  if (typeof window === "undefined") return "dark";
  const stored = localStorage.getItem("fas_landing_theme");
  if (stored === "dark" || stored === "light") return stored;
  return document.documentElement.dataset.theme === "light" ? "light" : "dark";
}

const mode = ref(initialMode());
const nextMode = computed(() => mode.value === "dark" ? "light" : "dark");
const entryRoute = computed(() => auth.isAuthenticated ? "/dashboard" : "/login");

function toggleMode() {
  mode.value = nextMode.value;
  localStorage.setItem("fas_landing_theme", mode.value);
}

let pressTimer = 0;
let pointerFrame = 0;
let sceneContext = null;
let sceneMatchMedia = null;
let smoothScroll = null;
let scrollHost = null;
let tickerCallback = null;
let mediaQuery = null;
let destroyed = false;
let refreshFrame = 0;

function pressScene() {
  activeFeature.value = (activeFeature.value + 1) % 4;
  if (reducedMotion.value) return;
  turns.value += 1;
  pressed.value = false;
  clearTimeout(pressTimer);
  requestAnimationFrame(() => {
    if (destroyed) return;
    pressed.value = true;
    pressTimer = window.setTimeout(() => (pressed.value = false), 850);
  });
}

function moveScene(event) {
  if (reducedMotion.value || event.pointerType === "touch") return;
  const rect = event.currentTarget.getBoundingClientRect();
  if (!rect.width || !rect.height) return;
  const x = Math.max(-1, Math.min(1, ((event.clientX - rect.left) / rect.width) * 2 - 1));
  const y = Math.max(-1, Math.min(1, ((event.clientY - rect.top) / rect.height) * 2 - 1));
  cancelAnimationFrame(pointerFrame);
  pointerFrame = requestAnimationFrame(() => {
    stage.value?.style.setProperty("--rx", `${-y * 8}deg`);
    stage.value?.style.setProperty("--ry", `${x * 12}deg`);
  });
}

function resetScene() {
  cancelAnimationFrame(pointerFrame);
  stage.value?.style.setProperty("--rx", "0deg");
  stage.value?.style.setProperty("--ry", "0deg");
}

function selectFeature(index) {
  activeFeature.value = index;
}

// Sumber masalah yang umum: layout Vue bisa punya wrapper overflow:auto.
// ScrollTrigger harus mendengarkan elemen yang SEBENARNYA digulir.
function findScrollHost(root) {
  for (let node = root?.parentElement; node && node !== document.documentElement; node = node.parentElement) {
    const css = getComputedStyle(node);
    if (/(auto|scroll)/.test(css.overflowY) && node.scrollHeight > node.clientHeight + 3) {
      return node;
    }
  }
  return window;
}

function scrollerHeight() {
  return scrollHost && scrollHost !== window ? scrollHost.clientHeight : window.innerHeight;
}

function scrollTop() {
  return scrollHost && scrollHost !== window ? scrollHost.scrollTop : window.scrollY;
}

function elementScrollPosition(element) {
  if (!element) return 0;
  const hostTop = scrollHost && scrollHost !== window
    ? scrollHost.getBoundingClientRect().top
    : 0;
  return element.getBoundingClientRect().top - hostTop + scrollTop();
}

function scrollToSection(id) {
  const root = landingRoot.value;
  if (!root) return;
  const section = id === "top" ? root : root.querySelector(`#${id}`);
  if (!section) return;

  // Menu mengarah KE AKHIR proses penyusunan, bukan sekadar ke awal section.
  const trigger = ScrollTrigger.getById(`fas-${id}`);
  const destination = id === "top"
    ? 0
    : trigger && Number.isFinite(trigger.start) && Number.isFinite(trigger.end)
      ? trigger.start + (trigger.end - trigger.start) * 0.86
      : elementScrollPosition(section);

  if (!reducedMotion.value && smoothScroll) {
    smoothScroll.scrollTo(destination, {
      duration: 2.6,
      easing: t => 1 - Math.pow(1 - t, 4),
      lock: false,
    });
  } else if (scrollHost && scrollHost !== window) {
    scrollHost.scrollTo({ top: destination, behavior: "auto" });
  } else {
    window.scrollTo({ top: destination, behavior: "auto" });
  }

  if (id === "top") {
    history.replaceState(history.state, "", `${location.pathname}${location.search}`);
  } else {
    history.replaceState(history.state, "", `#${id}`);
  }
}

function queueRefresh() {
  if (refreshFrame || destroyed) return;
  refreshFrame = requestAnimationFrame(() => {
    refreshFrame = 0;
    smoothScroll?.resize();
    ScrollTrigger.refresh(true);
  });
}

function createSmoothScroll() {
  if (reducedMotion.value) return;
  const wrapper = scrollHost === window ? window : scrollHost;
  const content = scrollHost === window
    ? document.documentElement
    : (scrollHost.firstElementChild || landingRoot.value);

  smoothScroll = new Lenis({
    wrapper,
    content,
    autoRaf: false,
    smoothWheel: true,
    syncTouch: false,
    duration: 1.75,
    wheelMultiplier: 0.82,
    touchMultiplier: 1,
    easing: t => Math.min(1, 1.001 - Math.pow(2, -10 * t)),
  });

  smoothScroll.on("scroll", ScrollTrigger.update);
  tickerCallback = time => smoothScroll?.raf(time * 1000);
  gsap.ticker.add(tickerCallback);
  // Petunjuk integrasi resmi Lenis + GSAP (scroll tetap native dan tidak dikunci).
  gsap.ticker.lagSmoothing(0);
}

const scatteredCards = [
  { x: -390, y: 180, rotation: -19, rotationY: -26, scale: 0.66 },
  { x: 250, y: -220, rotation: 18, rotationY: 32, scale: 0.72 },
  { x: -230, y: -170, rotation: -14, rotationY: -29, scale: 0.72 },
  { x: 370, y: 200, rotation: 21, rotationY: 35, scale: 0.66 },
];

function addScene(id, compose, { pin = true } = {}) {
  const section = landingRoot.value?.querySelector(`#${id}`);
  if (!section) return;

  // Saat dipin, 1.95 viewport dipakai khusus untuk animasi. Jadi beberapa
  // gerakan scroll diperlukan sebelum pindah ke slide berikutnya.
  const scrubLength = () => Math.max(1050, Math.round(scrollerHeight() * 1.95));
  const timeline = gsap.timeline({
    defaults: { ease: "none" },
    scrollTrigger: {
      id: `fas-${id}`,
      trigger: section,
      scroller: scrollHost === window ? undefined : scrollHost,
      start: pin ? "top top" : "top 90%",
      end: pin ? () => `+=${scrubLength()}` : "bottom 20%",
      pin,
      pinSpacing: true,
      pinReparent: false,
      anticipatePin: 1,
      scrub: pin ? 0.42 : 0.55,
      invalidateOnRefresh: true,
      refreshPriority: 0,
      markers: false,
    },
  });

  compose(timeline, section);
  // Ada fase hold yang mempertahankan semua objek di posisi akhirnya.
  // gsap.to({}) tak mengubah layout dan tetap menambah durasi timeline.
  timeline.to({}, { duration: 0.34 });
  return timeline;
}

function chapterPlatform(timeline, section) {
  const eyebrow = section.querySelector(".platform-stage header .section-kicker");
  const headline = section.querySelector(".platform-stage header h2");
  const subtitle = section.querySelector(".platform-stage header > p:last-child");
  const cards = gsap.utils.toArray(section.querySelectorAll(".feature-card"));

  timeline.fromTo(eyebrow,
    { x: -180, y: -110, rotation: -12, opacity: 0 },
    { x: 0, y: 0, rotation: 0, opacity: 1, duration: 0.65 }, 0.05);
  timeline.fromTo(headline,
    { x: -340, y: 140, rotation: -13, scale: 0.64, opacity: 0, filter: "blur(12px)" },
    { x: 0, y: 0, rotation: 0, scale: 1, opacity: 1, filter: "blur(0px)", duration: 0.95 }, 0.12);
  timeline.fromTo(subtitle,
    { x: 260, y: -95, rotation: 9, opacity: 0 },
    { x: 0, y: 0, rotation: 0, opacity: 1, duration: 0.7 }, 0.24);
  timeline.fromTo(cards,
    {
      x: i => scatteredCards[i].x,
      y: i => scatteredCards[i].y,
      rotation: i => scatteredCards[i].rotation,
      rotationY: i => scatteredCards[i].rotationY,
      scale: i => scatteredCards[i].scale,
      opacity: 0.05,
      filter: "blur(12px)",
      transformPerspective: 1100,
      transformOrigin: "center center",
    },
    {
      x: 0, y: 0, rotation: 0, rotationY: 0,
      scale: 1, opacity: 1, filter: "blur(0px)",
      duration: 1.42, stagger: 0.11,
    }, 0.27);
}

function chapterWorkflow(timeline, section) {
  const text = [...section.querySelectorAll(".workflow-stage > div:first-child > *")];
  const steps = [...section.querySelectorAll(".workflow-stage li")];
  timeline.fromTo(text,
    { x: -220, y: i => i % 2 ? 115 : -100, rotation: -8, opacity: 0, filter: "blur(8px)" },
    { x: 0, y: 0, rotation: 0, opacity: 1, filter: "blur(0px)", duration: 1.12, stagger: 0.1 }, 0.05);
  timeline.fromTo(steps,
    {
      x: i => [340, -275, 390][i],
      y: i => [-150, 170, 110][i],
      rotation: i => [16, -13, 19][i],
      rotationY: i => [-29, 25, -30][i],
      scale: 0.7, opacity: 0.04, filter: "blur(9px)",
    },
    {
      x: 0, y: 0, rotation: 0, rotationY: 0,
      scale: 1, opacity: 1, filter: "blur(0px)", duration: 1.18, stagger: 0.12,
    }, 0.22);
}

function chapterCompany(timeline, section) {
  const visual = section.querySelector(".company-visual");
  const text = [...section.querySelectorAll(".company-copy > *")];
  timeline.fromTo(visual,
    { x: -420, y: 140, rotation: -21, rotationY: -60, scale: 0.57, opacity: 0.05, filter: "blur(12px)" },
    { x: 0, y: 0, rotation: 0, rotationY: 0, scale: 1, opacity: 1, filter: "blur(0px)", duration: 1.5 }, 0.03);
  timeline.fromTo(text,
    {
      x: i => i % 2 === 0 ? 340 : 240,
      y: i => i % 2 === 0 ? -135 : 155,
      rotation: i => i % 2 === 0 ? 12 : -9,
      scale: 0.78, opacity: 0, filter: "blur(9px)",
    },
    { x: 0, y: 0, rotation: 0, scale: 1, opacity: 1, filter: "blur(0px)", duration: 1.35, stagger: 0.11 }, 0.18);
}

function chapterFinale(timeline, section) {
  const eyebrow = section.querySelector(".finale-kicker");
  const lines = [...section.querySelectorAll(".finale-line")];
  const subtitle = section.querySelector(".finale-description");
  const button = section.querySelector(".finale-action");
  timeline.fromTo(eyebrow,
    { y: -140, scale: 0.65, rotation: -8, opacity: 0 },
    { y: 0, scale: 1, rotation: 0, opacity: 1, duration: 0.76 }, 0);
  timeline.fromTo(lines,
    { x: i => i ? 500 : -500, y: i => i ? -155 : 150, rotation: i => i ? 13 : -13, scale: 0.62, opacity: 0, filter: "blur(13px)" },
    { x: 0, y: 0, rotation: 0, scale: 1, opacity: 1, filter: "blur(0px)", duration: 1.4, stagger: 0.17 }, 0.14);
  timeline.fromTo(subtitle,
    { y: 125, opacity: 0, filter: "blur(8px)" },
    { y: 0, opacity: 1, filter: "blur(0px)", duration: 1.02 }, 0.57);
  timeline.fromTo(button,
    { y: 145, scale: 0.75, opacity: 0 },
    { y: 0, scale: 1, opacity: 1, duration: 0.95 }, 0.7);
}

function setupScenes() {
  const root = landingRoot.value;
  if (!root || reducedMotion.value) return;
  sceneContext = gsap.context(() => {
    const hero = root.querySelector(".landing-hero");
    const intro = [...root.querySelectorAll(".hero-copy > *")];
    const sculpture = root.querySelector(".hero-sculpture");

    gsap.fromTo(intro,
      { x: -40, y: 65, opacity: 0, filter: "blur(8px)" },
      { x: 0, y: 0, opacity: 1, filter: "blur(0px)", duration: 1.18, stagger: 0.14, ease: "power3.out" });
    gsap.fromTo(sculpture,
      { x: 150, y: 95, rotation: 11, rotationY: -26, scale: 0.72, opacity: 0 },
      { x: 0, y: 0, rotation: 0, rotationY: 0, scale: 1, opacity: 1, duration: 1.55, ease: "power3.out", delay: 0.2 });

    // Saat slide hero menjauh, teks dan objek F bergerak ke arah berbeda.
    gsap.timeline({ scrollTrigger: {
      id: "fas-hero",
      trigger: hero,
      scroller: scrollHost === window ? undefined : scrollHost,
      start: "top top",
      end: "bottom top",
      scrub: 0.45,
    } })
      .to(root.querySelector(".hero-copy"), { x: -90, y: -110, scale: 0.88, opacity: 0.1, ease: "none" }, 0)
      .to(sculpture, { x: 140, y: 110, scale: 1.07, rotationY: 17, opacity: 0.1, ease: "none" }, 0);

    sceneMatchMedia = gsap.matchMedia();
    sceneMatchMedia.add({
      desktop: "(min-width: 1100px) and (min-height: 720px)",
      compact: "(max-width: 1099px), (max-height: 719px)",
    }, context => {
      const pin = Boolean(context.conditions.desktop);
      addScene("platform", chapterPlatform, { pin });
      addScene("workflow", chapterWorkflow, { pin });
      addScene("company", chapterCompany, { pin });
      addScene("start", chapterFinale, { pin });
    });
  }, root);
}

function stopMotion() {
  sceneMatchMedia?.revert();
  sceneMatchMedia = null;
  sceneContext?.revert();
  sceneContext = null;
  if (tickerCallback) gsap.ticker.remove(tickerCallback);
  tickerCallback = null;
  smoothScroll?.destroy();
  smoothScroll = null;
}

function synchronizeMotion() {
  if (destroyed) return;
  stopMotion();
  reducedMotion.value = Boolean(mediaQuery?.matches);
  if (reducedMotion.value) {
    gsap.set(landingRoot.value?.querySelectorAll(".scroll-chapter *") || [], {
      clearProps: "transform,opacity,filter,visibility",
    });
    return;
  }
  createSmoothScroll();
  setupScenes();
  queueRefresh();
}

onMounted(async () => {
  await nextTick();
  if (destroyed) return;
  scrollHost = findScrollHost(landingRoot.value);
  mediaQuery = window.matchMedia("(prefers-reduced-motion: reduce)");
  mediaQuery.addEventListener?.("change", synchronizeMotion);
  synchronizeMotion();

  // Font web dan gambar dapat menggeser pengukuran. Refresh sesudah render.
  document.fonts?.ready.then(queueRefresh);
  // ScrollTrigger sudah melakukan refresh otomatis saat ukuran viewport berubah.
  // Hindari ResizeObserver pada root: pinSpacing bisa memicu refresh berulang.

  // Akses langsung lewat /#company tetap berada di section yang benar.
  requestAnimationFrame(() => {
    if (destroyed) return;
    const id = location.hash.slice(1);
    if (["platform", "workflow", "company", "start"].includes(id)) {
      scrollToSection(id);
    }
    if (import.meta.env.DEV) {
      console.info("[FAS motion] aktif", {
        scroller: scrollHost === window ? "window" : "container aplikasi",
        reducedMotion: reducedMotion.value,
        sceneCount: ScrollTrigger.getAll().filter(item => item.vars.id?.startsWith("fas-")).length,
      });
    }
  });
});

onBeforeUnmount(() => {
  destroyed = true;
  clearTimeout(pressTimer);
  cancelAnimationFrame(pointerFrame);
  cancelAnimationFrame(refreshFrame);
  mediaQuery?.removeEventListener?.("change", synchronizeMotion);
  stopMotion();
});

const iconPaths = [
  "M8 4h8v4H8z M6 6H4v15h16V6h-2 M8 12h8 M8 16h5",
  "M3 7l9-4 9 4-9 4z M3 7v10l9 4 9-4V7 M12 11v10",
  "M3 6h11v12H3z M14 10h4l3 4v4h-7 M7 18a2 2 0 1 0 0 .1 M18 18a2 2 0 1 0 0 .1",
  "M3 5h18v15H3z M3 9h18 M15 14h3 M6 14h3",
];
const features = [
  {
    number: "01",
    title: "Operations",
    description:
      "Dari pesanan customer hingga persiapan barang, setiap tahap lebih mudah ditelusuri.",
    tag: "Pesanan · Gudang",
  },
  {
    number: "02",
    title: "Inventory",
    description:
      "Pantau ketersediaan produk, stok minimum, dan kebutuhan gudang dalam satu tempat.",
    tag: "Produk · Persediaan",
  },
  {
    number: "03",
    title: "Distribution",
    description:
      "Hubungkan proses picking, packing, dan pengiriman dengan dokumen yang sesuai.",
    tag: "Picking · Packing · Pengiriman",
  },
  {
    number: "04",
    title: "Finance",
    description:
      "Lihat faktur, piutang, pembayaran, dan laporan untuk keputusan perusahaan.",
    tag: "Transaksi · Laporan",
  },
];
</script>
<template>
  <main ref="landingRoot" class="fas-landing" :data-landing-theme="mode">
    <div class="landing-glow glow-a"></div>
    <div class="landing-glow glow-b"></div>
    <header class="landing-nav">
      <RouterLink to="/" class="landing-brand"
        ><span class="brand-symbol">F<span>✦</span></span>
        <div>
          <strong>PT. FAS</strong><small>FURI ANUGERAH SEJAHTERA</small>
        </div></RouterLink
      >
      <nav aria-label="Navigasi beranda">
        <a href="#platform" @click.prevent="scrollToSection('platform')"
          >Ekosistem</a
        ><a href="#workflow" @click.prevent="scrollToSection('workflow')"
          >Alur kerja</a
        ><a href="#company" @click.prevent="scrollToSection('company')"
          >Tentang FAS</a
        >
      </nav>
      <div class="nav-actions">
        <button
          type="button"
          class="theme-switch"
          :aria-label="
            nextMode === 'light'
              ? 'Aktifkan mode terang'
              : 'Aktifkan mode gelap'
          "
          :title="nextMode === 'light' ? 'Mode terang' : 'Mode gelap'"
          @click="toggleMode"
        >
          <svg
            v-if="mode === 'dark'"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.7"
            stroke-linecap="round"
            aria-hidden="true"
          >
            <circle cx="12" cy="12" r="4" />
            <path
              d="M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.5 1.5m11 11L19 19M5 19l1.5-1.5m11-11L19 5"
            />
          </svg>
          <svg
            v-else
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.7"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
          >
            <path d="M21 13A9 9 0 0 1 11 3a9 9 0 1 0 10 10Z" />
          </svg>
        </button>
        <RouterLink
          class="landing-enter"
          :to="entryRoute"
          >{{ auth.isAuthenticated ? "Buka dashboard" : "Masuk sistem"
          }}<span aria-hidden="true">↗︎</span></RouterLink
        >
      </div>
    </header>
    <section class="landing-hero">
      <div class="hero-copy">
        <p class="hero-kicker"><span></span>PT FURI ANUGERAH SEJAHTERA</p>
        <h1>
          Distribusi terarah.<br />Bisnis bergerak.<br /><em>Bersama FAS.</em>
        </h1>
        <p class="hero-description">
          Menghubungkan operasional, persediaan, distribusi, dan keuangan. Satu
          ekosistem untuk langkah perusahaan yang lebih teratur, dari pesanan
          pertama hingga laporan terakhir.
        </p>
        <div class="hero-actions">
          <RouterLink
            class="hero-primary"
            :to="entryRoute"
            >{{
              auth.isAuthenticated ? "Lanjutkan pekerjaan" : "Masuk ke PT. FAS"
            }}<span aria-hidden="true">↗︎</span></RouterLink
          ><a
            href="#platform"
            @click.prevent="scrollToSection('platform')"
            class="hero-secondary"
            >Jelajahi platform <span aria-hidden="true">↓</span></a
          >
        </div>
        <div class="hero-footnote">
          <span class="tiny-cross">+</span>
          <p>
            Dirancang untuk kerja tim.<br /><strong
              >Operations · Inventory · Distribution · Finance.</strong
            >
          </p>
        </div>
      </div>
      <div
        class="hero-sculpture"
        @pointermove="moveScene"
        @pointerleave="resetScene"
      >
        <div class="sculpture-aura" aria-hidden="true"></div>
        <button
          ref="stage"
          :style="{ '--spin': `${turns * 360}deg` }"
          type="button"
          :class="['sculpture-stage', { pressed }]"
          aria-label="Putar animasi 3D FAS dan jelajahi modul berikutnya"
          @click="pressScene"
        >
          <span class="sculpture-shadow" aria-hidden="true"></span>
          <span class="sculpture-float" aria-hidden="true">
            <span class="sculpture-ring ring-a"></span
            ><span class="sculpture-ring ring-b"></span
            ><span class="sculpture-ring ring-c"></span>
            <span class="fas-gem"
              ><span class="gem-back"></span><span class="gem-side"></span
              ><span class="gem-front">F<span>✦</span></span
              ><span class="gem-shine"></span
            ></span>
            <span class="satellite satellite-a"></span
            ><span class="satellite satellite-b"></span
            ><span class="satellite satellite-c"></span>
          </span>
          <span
            v-for="(item, index) in features"
            :key="item.number"
            :class="[
              'sculpture-label',
              'label-' + index,
              { selected: activeFeature === index },
            ]"
            aria-hidden="true"
            ><svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="1.6"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path :d="iconPaths[index]" /></svg
            >{{ item.title }}</span
          >
        </button>
        <div class="sculpture-caption" aria-live="polite">
          <span class="caption-line"></span>
          <p>
            <strong>{{ features[activeFeature].title }}</strong
            ><span>{{ features[activeFeature].tag }}</span>
          </p>
          <span class="scene-control-hint"
            >Sentuh untuk bergerak <span aria-hidden="true">↗︎</span></span
          >
        </div>
      </div>
    </section>
    <div class="module-ribbon" aria-hidden="true">
      <span>OPERATIONS</span><i>✦</i><span>INVENTORY</span><i>✦</i
      ><span>DISTRIBUTION</span><i>✦</i><span>FINANCE</span><i>✦</i>
    </div>
    <section id="platform" class="platform-section scroll-chapter">
      <div class="scene-stage platform-stage">
        <header>
          <p class="section-kicker">ONE CONNECTED WORKSPACE</p>
          <h2>Setiap bagian.<br />Satu arah yang sama.</h2>
          <p>
            Alur yang saling terhubung, dari barang hingga pencatatan keuangan.
          </p>
        </header>
        <div class="platform-grid">
          <button
            v-for="(item, index) in features"
            :key="item.number"
            type="button"
            :class="['feature-card', { chosen: activeFeature === index }]"
            :aria-pressed="activeFeature === index"
            @click="selectFeature(index)"
          >
            <div class="feature-top">
              <span class="feature-icon"
                ><svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  aria-hidden="true"
                >
                  <path :d="iconPaths[Number(item.number) - 1]" /></svg></span
              ><span>{{ item.number }} ↗︎</span>
            </div>
            <h3>{{ item.title }}</h3>
            <p>{{ item.description }}</p>
            <small>{{ item.tag }} <span aria-hidden="true">→</span></small>
          </button>
        </div>
      </div>
    </section>
    <section id="workflow" class="workflow-section scroll-chapter">
      <div class="scene-stage workflow-stage">
        <div>
          <p class="section-kicker">BUILT AROUND YOUR TEAM</p>
          <h2>Pekerjaan lebih rapi.<br />Informasi lebih berarti.</h2>
          <p>
            Akun pribadi dan hak akses sesuai peran membantu setiap anggota tim
            fokus pada tugasnya.
          </p>
          <RouterLink
            :to="entryRoute"
            class="workflow-link"
            >Buka ruang kerja <span aria-hidden="true">→</span></RouterLink
          >
        </div>
        <ol>
          <li>
            <span>01</span>
            <div>
              <h3>Pesanan dicatat</h3>
              <p>Data customer dan kebutuhan barang berada dalam satu alur.</p>
            </div>
          </li>
          <li>
            <span>02</span>
            <div>
              <h3>Barang diproses</h3>
              <p>Tim gudang melanjutkan persiapan dan distribusi.</p>
            </div>
          </li>
          <li>
            <span>03</span>
            <div>
              <h3>Transaksi ditelusuri</h3>
              <p>Finance memantau faktur, pembayaran, dan laporan.</p>
            </div>
          </li>
        </ol>
      </div>
    </section>
    <section id="company" class="company-section scroll-chapter">
      <div class="scene-stage company-stage">
        <div class="company-visual"><PharmaceuticalScene compact /></div>
        <div class="company-copy">
          <p class="section-kicker">CARE IN EVERY CONNECTION</p>
          <h2>PT Furi<br />Anugerah Sejahtera.</h2>
          <p>
            Di balik setiap pesanan, ada tim yang saling mengandalkan. FAS
            menyatukan informasi agar pekerjaan gudang, distribusi, dan finance
            dapat dilanjutkan dengan lebih mudah.
          </p>
          <div class="company-values">
            <span>Akses sesuai peran</span><span>Dokumen terhubung</span
            ><span>Informasi terpusat</span>
          </div>
          <RouterLink
            :to="entryRoute"
            class="hero-primary"
            >{{ auth.isAuthenticated ? "Buka Overview" : "Login ke ruang kerja"
            }}<span aria-hidden="true">↗︎</span></RouterLink
          >
        </div>
      </div>
    </section>

    <!-- PENUTUP: satu slide di tengah (bukan footer overlay atau fake sticky). -->
    <section id="start" class="finale-section scroll-chapter">
      <div class="scene-stage finale-stage">
        <span class="finale-halo halo-one" aria-hidden="true"></span>
        <span class="finale-halo halo-two" aria-hidden="true"></span>
        <p class="finale-kicker"><span class="finale-kicker-dot"></span> PT FURI ANUGERAH SEJAHTERA</p>
        <h2 class="finale-heading">
          <span class="finale-line">Satu perusahaan.</span>
          <span class="finale-line finale-accent">Semua terhubung.</span>
        </h2>
        <p class="finale-description">Dari pesanan, persediaan, pengiriman, hingga laporan keuangan. Seluruh tim bergerak dalam satu ekosistem.</p>
        <RouterLink class="hero-primary finale-action" :to="entryRoute">
          {{ auth.isAuthenticated ? 'Buka dashboard' : 'Masuk ke ruang kerja' }}
          <span aria-hidden="true">↗︎</span>
        </RouterLink>
      </div>
    </section>
    <footer class="landing-footer">
      <strong>PT. FAS</strong
      ><span>Furi Anugerah Sejahtera · Business Operations Platform</span
      ><a
        href="#"
        @click.prevent="scrollToSection('top')"
        aria-label="Kembali ke atas"
        >Kembali ke atas ↑</a
      >
    </footer>
  </main>
</template>
<style scoped>
.fas-landing {
  position: relative;
  isolation: isolate;
  overflow: clip;
  color: var(--fas-text);
  background: linear-gradient(
    150deg,
    var(--fas-bg),
    color-mix(in srgb, var(--fas-accent) 7%, var(--fas-surface))
  );
  min-height: 100vh;
  font-family: inherit;
}
.landing-glow {
  position: absolute;
  z-index: -1;
  border-radius: 50%;
  filter: blur(90px);
  pointer-events: none;
}
.glow-a {
  width: 700px;
  height: 700px;
  top: -200px;
  right: -200px;
  background: var(--fas-accent-soft);
}
.glow-b {
  width: 450px;
  height: 450px;
  top: 700px;
  left: -250px;
  background: var(--fas-accent-soft);
}
.landing-nav,
.landing-hero,
.platform-section,
.workflow-section,
.landing-footer {
  width: min(1240px, calc(100% - 96px));
  margin: auto;
}
.landing-nav {
  height: 110px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}
.landing-brand {
  display: flex;
  align-items: center;
  gap: 12px;
  text-decoration: none;
  color: var(--fas-text);
}
.brand-symbol {
  position: relative;
  display: grid;
  place-items: center;
  width: 44px;
  height: 44px;
  border: 1px solid var(--fas-accent-border);
  background: var(--fas-accent-soft);
  border-radius: 15px;
  color: var(--fas-accent-text);
  font-size: 27px;
  font-weight: 800;
}
.brand-symbol > span {
  position: absolute;
  right: -4px;
  bottom: -8px;
  font-size: 20px;
}
.landing-brand strong {
  font-size: 18px;
  letter-spacing: -0.04em;
}
.landing-brand small {
  display: block;
  font-size: 8px;
  letter-spacing: 0.16em;
  color: var(--fas-muted);
  margin-top: 3px;
}
.landing-nav nav {
  display: flex;
  gap: 30px;
  font-size: 12px;
  color: var(--fas-muted);
}
.landing-nav nav a:hover {
  color: var(--fas-accent-text);
}
.landing-enter {
  border: 1px solid var(--fas-border-strong);
  border-radius: 50px;
  padding: 13px 20px;
  display: flex;
  gap: 25px;
  font-size: 12px;
  font-weight: 650;
  color: var(--fas-text);
  background: var(--fas-card);
}
.landing-hero {
  display: grid;
  grid-template-columns: 1fr 1fr;
  align-items: center;
  min-height: 650px;
  gap: 20px;
  padding: 30px 0 75px;
}
.hero-kicker {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 9px;
  letter-spacing: 0.15em;
  font-weight: 700;
  color: var(--fas-accent-text);
  margin-bottom: 25px;
}
.hero-kicker > span {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--fas-accent);
}
.hero-copy h1 {
  font-size: clamp(42px, 5.3vw, 74px);
  line-height: 1.06;
  letter-spacing: -0.065em;
  font-weight: 700;
  margin: 0;
}
.hero-copy h1 em {
  font-style: normal;
  color: var(--fas-accent-text);
}
.hero-description {
  max-width: 410px;
  font-size: 14px;
  line-height: 1.9;
  color: var(--fas-muted);
  margin: 25px 0 30px;
}
.hero-actions {
  display: flex;
  gap: 22px;
  align-items: center;
  flex-wrap: wrap;
}
.hero-primary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 28px;
  border-radius: 50px;
  background: var(--fas-accent);
  color: #fff;
  padding: 17px 24px;
  font-size: 12px;
  font-weight: 700;
  box-shadow: 0 12px 25px var(--fas-accent-soft);
}
.hero-primary:hover {
  background: var(--fas-accent-hover);
}
.hero-secondary {
  font-size: 11px;
  font-weight: 600;
  color: var(--fas-text-soft);
}
.hero-secondary span {
  margin-left: 10px;
}
.hero-footnote {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 40px;
  color: var(--fas-muted);
  font-size: 10px;
  line-height: 1.8;
}
.tiny-cross {
  font-size: 32px;
  font-weight: 300;
  color: var(--fas-accent-text);
}
.hero-footnote strong {
  color: var(--fas-text-soft);
  font-weight: 500;
}
.hero-art {
  position: relative;
  border: 1px solid var(--fas-accent-border);
  border-radius: 46% 46% 28px 28px;
  background: radial-gradient(
    ellipse at 45% 35%,
    var(--fas-accent-soft),
    transparent 75%
  );
  padding-top: 40px;
}
.art-label {
  display: block;
  text-align: center;
  font-size: 8px;
  letter-spacing: 0.2em;
  color: var(--fas-accent-text);
  font-weight: 600;
}
.art-footer {
  display: flex;
  justify-content: space-between;
  padding: 20px 28px;
  border-top: 1px solid var(--fas-border);
  font-size: 8px;
  letter-spacing: 0.12em;
  color: var(--fas-muted);
}
.platform-section {
  padding: 65px 0;
  border-top: 1px solid var(--fas-border);
}
.platform-section header {
  display: grid;
  grid-template-columns: 1fr 1fr;
  align-items: end;
  gap: 12px;
}
.section-kicker {
  font-size: 9px;
  letter-spacing: 0.18em;
  color: var(--fas-accent-text);
  font-weight: 700;
  grid-column: 1/-1;
}
.platform-section h2,
.workflow-section h2 {
  font-size: clamp(28px, 3vw, 40px);
  line-height: 1.15;
  letter-spacing: -0.04em;
  font-weight: 650;
  margin: 10px 0;
}
.platform-section header > p:last-child {
  font-size: 13px;
  color: var(--fas-muted);
  line-height: 1.8;
  max-width: 350px;
  justify-self: end;
}
.platform-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 20px;
  margin-top: 35px;
}
.platform-grid .feature-card {
  padding: 28px;
  border: 1px solid var(--fas-border);
  border-radius: 22px;
  background: var(--fas-card);
  transition:
    transform 0.25s,
    border-color 0.25s;
  box-shadow: var(--fas-shadow-soft);
}
.platform-grid .feature-card:hover {
  transform: translateY(-5px);
  border-color: var(--fas-accent-border);
}
.feature-top {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  color: var(--fas-accent-text);
}
.platform-grid h3 {
  font-size: 23px;
  letter-spacing: -0.03em;
  font-weight: 600;
  margin: 28px 0 12px;
}
.platform-grid p {
  font-size: 12px;
  color: var(--fas-muted);
  line-height: 1.9;
  min-height: 70px;
}
.platform-grid small {
  display: block;
  border-top: 1px solid var(--fas-border);
  padding-top: 18px;
  margin-top: 25px;
  font-size: 9px;
  color: var(--fas-accent-text);
}
.workflow-section {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 70px;
  padding: 55px 0 85px;
}
.workflow-section > div > p:not(.section-kicker) {
  font-size: 13px;
  line-height: 1.9;
  color: var(--fas-muted);
  max-width: 350px;
}
.workflow-link {
  display: inline-flex;
  gap: 30px;
  font-size: 12px;
  color: var(--fas-accent-text);
  font-weight: 650;
  margin-top: 20px;
}
.workflow-section ol {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: 0;
}
.workflow-section li {
  display: flex;
  gap: 20px;
  padding: 22px 0;
  border-bottom: 1px solid var(--fas-border);
}
.workflow-section li > span {
  font-size: 11px;
  color: var(--fas-accent-text);
  padding-top: 5px;
}
.workflow-section h3 {
  font-size: 16px;
  font-weight: 650;
  margin: 0;
}
.workflow-section li p {
  font-size: 11px;
  line-height: 1.8;
  color: var(--fas-muted);
  margin: 7px 0 0;
}
.landing-footer {
  border-top: 1px solid var(--fas-border);
  display: flex;
  gap: 20px;
  justify-content: space-between;
  align-items: center;
  padding: 30px 0;
  font-size: 9px;
  color: var(--fas-muted);
}
.landing-footer > strong {
  font-size: 16px;
  color: var(--fas-text);
  letter-spacing: -0.04em;
}
.fas-landing :is(a, button):focus-visible {
  outline: 2px solid var(--fas-accent);
  outline-offset: 5px;
}
@media (max-width: 900px) {
  .landing-nav,
  .landing-hero,
  .platform-section,
  .workflow-section,
  .landing-footer {
    width: calc(100% - 48px);
  }
  .landing-hero {
    grid-template-columns: 1fr;
    gap: 45px;
  }
  .hero-art {
    max-width: 570px;
    width: 100%;
    justify-self: center;
  }
  .hero-copy {
    padding-top: 25px;
  }
  .hero-copy h1 {
    font-size: clamp(44px, 8vw, 68px);
  }
  .platform-grid {
    gap: 12px;
  }
  .platform-grid .feature-card {
    padding: 20px;
  }
  .workflow-section {
    gap: 30px;
  }
  .landing-nav nav {
    display: none;
  }
}
@media (max-width: 600px) {
  .landing-nav,
  .landing-hero,
  .platform-section,
  .workflow-section,
  .landing-footer {
    width: calc(100% - 36px);
  }
  .landing-nav {
    height: 85px;
  }
  .landing-brand small {
    font-size: 6px;
  }
  .landing-enter {
    font-size: 10px;
    padding: 11px 14px;
    gap: 12px;
  }
  .landing-hero {
    padding-bottom: 45px;
    min-height: 0;
  }
  .hero-copy h1 {
    font-size: 44px;
  }
  .hero-description {
    font-size: 12px;
  }
  .hero-art {
    padding-top: 28px;
  }
  .hero-actions {
    gap: 18px;
  }
  .hero-primary {
    font-size: 11px;
    padding: 15px 20px;
  }
  .platform-section {
    padding: 40px 0;
  }
  .platform-section header,
  .platform-grid,
  .workflow-section {
    grid-template-columns: 1fr;
  }
  .platform-section header > p:last-child {
    justify-self: start;
  }
  .platform-grid p {
    min-height: 0;
  }
  .workflow-section {
    padding: 35px 0 50px;
    gap: 15px;
  }
  .landing-footer {
    flex-wrap: wrap;
    line-height: 1.8;
  }
  .landing-footer > span {
    max-width: 180px;
  }
  .hero-footnote {
    margin-top: 25px;
  }
}
@media (prefers-reduced-motion: reduce) {
  .platform-grid .feature-card {
    transition: none;
  }
}
.module-ribbon {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 28px;
  padding: 25px 18px;
  border-block: 1px solid var(--fas-border);
  background: var(--fas-overlay);
  color: var(--fas-text-soft);
  font-size: 11px;
  letter-spacing: 0.15em;
}
.module-ribbon i {
  font-style: normal;
  color: var(--fas-accent-text);
}
.feature-icon {
  display: grid;
  place-items: center;
  border: 1px solid var(--fas-accent-border);
  border-radius: 12px;
  background: var(--fas-accent-soft);
  width: 44px;
  height: 44px;
}
.feature-icon svg {
  width: 23px;
  height: 23px;
}
.feature-top {
  align-items: center;
}
.feature-top > span:last-child {
  font-size: 10px;
  color: var(--fas-muted);
}
.company-section {
  width: min(1240px, calc(100% - 96px));
  margin: 0 auto 80px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  align-items: center;
  gap: 65px;
  border-top: 1px solid var(--fas-border);
  padding-top: 55px;
}
.company-visual {
  border-radius: 36px;
  background: radial-gradient(
    ellipse at center,
    var(--fas-accent-soft),
    transparent 70%
  );
}
.company-copy h2 {
  font-size: clamp(32px, 3.8vw, 50px);
  line-height: 1.12;
  font-weight: 650;
  letter-spacing: -0.05em;
  margin: 18px 0;
}
.company-copy > p:not(.section-kicker) {
  color: var(--fas-muted);
  font-size: 13px;
  line-height: 1.9;
}
.company-values {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 24px 0;
}
.company-values span {
  padding: 8px 10px;
  border: 1px solid var(--fas-border);
  border-radius: 8px;
  font-size: 9px;
  color: var(--fas-text-soft);
}
.company-copy > .hero-primary {
  width: fit-content;
}
.hero-copy {
  animation: copy-arrive 0.8s ease-out both;
}
.hero-primary,
.landing-enter {
  transition:
    transform 0.25s,
    box-shadow 0.25s;
}
.hero-primary:hover,
.landing-enter:hover {
  transform: translateY(-3px);
  box-shadow: 0 10px 30px var(--fas-accent-soft);
}
@keyframes workspace-arrive {
  from {
    opacity: 0;
    transform: perspective(1400px) rotateY(-12deg) rotateX(6deg)
      translateY(25px);
  }
  to {
    opacity: 1;
    transform: perspective(1400px) rotateY(-7deg) rotateX(3deg);
  }
}
@keyframes copy-arrive {
  from {
    opacity: 0;
    transform: translateY(18px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
@keyframes data-flow {
  from {
    left: -50px;
  }
  to {
    left: 105%;
  }
}
@keyframes module-breathe {
  0%,
  20%,
  100% {
    border-color: var(--fas-border);
    transform: translateY(0);
  }
  35%,
  55% {
    border-color: var(--fas-accent-border);
    transform: translateY(-4px);
    box-shadow: 0 8px 25px var(--fas-accent-soft);
  }
}
@keyframes bar-wave {
  from {
    transform: scaleY(0.65);
  }
  to {
    transform: scaleY(1);
  }
}
@keyframes star-turn {
  to {
    transform: rotate(360deg);
  }
}
@media (max-width: 1100px) {
  .platform-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .orbit-module {
    padding: 14px 10px;
    gap: 8px;
  }
  .ecosystem-hub strong {
    font-size: 20px;
  }
}
@media (max-width: 900px) {
  .ecosystem-art {
    transform: none;
    animation: copy-arrive 0.8s ease-out both;
  }
  .company-section {
    width: calc(100% - 48px);
    gap: 25px;
  }
  .hero-copy h1 {
    font-size: clamp(40px, 7vw, 66px);
  }
}
@media (max-width: 600px) {
  .platform-grid {
    grid-template-columns: 1fr;
  }
  .company-section {
    grid-template-columns: 1fr;
    width: calc(100% - 36px);
    margin-bottom: 45px;
    gap: 5px;
  }
  .company-copy {
    padding-bottom: 15px;
  }
  .orbit-modules {
    padding: 0 14px;
    gap: 8px;
  }
  .ecosystem-heading {
    padding: 18px 14px;
    font-size: 7px;
  }
  .ecosystem-hub {
    padding: 25px 16px;
    gap: 15px;
  }
  .ecosystem-hub > span {
    width: 60px;
    height: 60px;
  }
  .ecosystem-hub strong {
    font-size: 22px;
  }
  .ecosystem-bottom {
    margin: 18px 14px;
    gap: 12px;
    padding: 14px 10px;
  }
  .mini-chart {
    width: 65px;
  }
  .hero-copy h1 {
    font-size: clamp(35px, 8.7vw, 48px);
  }
  .hero-kicker {
    font-size: 8px;
  }
  .module-ribbon {
    gap: 13px;
    font-size: 8px;
  }
  .orbit-module strong {
    font-size: 12px;
  }
  .hero-footnote {
    font-size: 9px;
  }
}
@media (prefers-reduced-motion: reduce) {
  .fas-landing *,
  .fas-landing *::before,
  .fas-landing *::after {
    animation: none !important;
    transition: none !important;
  }
  .ecosystem-art {
    transform: none;
  }
}
/* Palet landing terpisah dari pilihan accent dashboard internal. */
.fas-landing[data-landing-theme="light"] {
  color-scheme: light;
  --fas-bg: #f4faf7;
  --fas-surface: #eef8f2;
  --fas-card: #fff;
  --fas-elevated: #f0f8f3;
  --fas-overlay: #e8f4ed;
  --fas-text: #102c22;
  --fas-text-soft: #34564a;
  --fas-muted: #607d70;
  --fas-border: #dbe9e1;
  --fas-border-strong: #c8ddd0;
  --fas-accent: #059669;
  --fas-accent-hover: #047857;
  --fas-accent-text: #047857;
  --fas-accent-soft: #05966914;
  --fas-accent-border: #05966938;
  --fas-shadow: #12372815;
  --fas-shadow-soft: 0 15px 35px #194e3210;
  --sculpture-blue: #35d5a1;
  --sculpture-purple: #047857;
  --sculpture-core: #0aab78;
  --sculpture-glass: #b0f2d7;
}
.fas-landing[data-landing-theme="dark"] {
  color-scheme: dark;
  --fas-bg: #05060c;
  --fas-surface: #0b0b19;
  --fas-card: #0e1020;
  --fas-elevated: #14162c;
  --fas-overlay: #ffffff04;
  --fas-text: #f1f1ff;
  --fas-text-soft: #c5cae7;
  --fas-muted: #929ab9;
  --fas-border: #b3b5ff19;
  --fas-border-strong: #b3b5ff2e;
  --fas-accent: #5868f2;
  --fas-accent-hover: #727df9;
  --fas-accent-text: #a2a7ff;
  --fas-accent-soft: #6973ff15;
  --fas-accent-border: #8481ff40;
  --fas-shadow: #00000044;
  --fas-shadow-soft: 0 15px 35px #00000022;
  --sculpture-blue: #4b9bff;
  --sculpture-purple: #a347f5;
  --sculpture-core: #5144cc;
  --sculpture-glass: #9083ff;
}
.fas-landing {
  transition:
    background 0.5s,
    color 0.5s;
  background:
    radial-gradient(
      ellipse at 85% 14%,
      color-mix(in srgb, var(--sculpture-purple) 9%, transparent),
      transparent 36%
    ),
    linear-gradient(145deg, var(--fas-bg), var(--fas-surface));
}
.fas-landing[data-landing-theme="dark"] {
  background:
    radial-gradient(ellipse at 92% 20%, #6030c21a, transparent 40%),
    radial-gradient(ellipse at 55% 3%, #2456cd16, transparent 35%), #05060c;
}
.glow-a {
  background: color-mix(in srgb, var(--sculpture-blue) 12%, transparent);
}
.glow-b {
  background: color-mix(in srgb, var(--sculpture-purple) 8%, transparent);
}
.nav-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}
.theme-switch {
  width: 45px;
  height: 45px;
  display: grid;
  place-items: center;
  flex-shrink: 0;
  border: 1px solid var(--fas-border-strong);
  border-radius: 50%;
  background: var(--fas-card);
  color: var(--fas-accent-text);
  cursor: pointer;
  transition:
    transform 0.35s,
    background 0.35s;
}
.theme-switch svg {
  width: 21px;
  height: 21px;
}
.theme-switch:hover {
  transform: rotate(18deg);
}
.theme-switch:active {
  transform: scale(0.9);
}
.hero-sculpture {
  position: relative;
  min-width: 0;
  align-self: center;
  perspective: 1100px;
}
.sculpture-aura {
  position: absolute;
  inset: 10%;
  border-radius: 50%;
  background: radial-gradient(
    circle at 25% 30%,
    color-mix(in srgb, var(--sculpture-blue) 22%, transparent),
    color-mix(in srgb, var(--sculpture-purple) 13%, transparent) 45%,
    transparent 70%
  );
  filter: blur(35px);
  pointer-events: none;
}
.sculpture-stage {
  --rx: 0deg;
  --ry: 0deg;
  position: relative;
  display: block;
  width: 100%;
  height: 490px;
  border: 0;
  padding: 0;
  background: transparent;
  cursor: pointer;
  perspective: 1000px;
  transform-style: preserve-3d;
  transform: rotateX(var(--rx)) rotateY(var(--ry));
  transition: transform 0.7s cubic-bezier(0.2, 0.8, 0.2, 1);
  -webkit-tap-highlight-color: transparent;
}
.sculpture-stage:focus-visible {
  outline: 2px solid var(--fas-accent);
  outline-offset: 5px;
  border-radius: 35px;
}
.sculpture-float {
  position: absolute;
  left: 50%;
  top: 48%;
  width: 230px;
  height: 230px;
  margin: -115px;
  transform-style: preserve-3d;
  animation: sculpture-hover 8s ease-in-out infinite;
}
.fas-gem {
  position: absolute;
  inset: 24px;
  transform-style: preserve-3d;
  transform: rotateX(-17deg) rotateY(calc(-25deg + var(--spin, 0deg)))
    rotateZ(-9deg);
  transition: transform 1.1s cubic-bezier(0.18, 0.8, 0.2, 1);
}
.gem-front,
.gem-back {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  border-radius: 38px;
}
.gem-front {
  background: linear-gradient(
    145deg,
    var(--sculpture-glass),
    var(--sculpture-core) 65%
  );
  border: 1px solid #ffffff65;
  color: #fff;
  font-size: 122px;
  line-height: 1;
  font-weight: 850;
  letter-spacing: -12px;
  transform: translateZ(30px);
  box-shadow:
    inset 2px 2px 5px #ffffff66,
    inset -4px -5px 15px #00000022;
  text-shadow: 0 5px 0 #00000016;
}
.gem-front > span {
  position: absolute;
  right: 13px;
  bottom: 15px;
  font-size: 39px;
  letter-spacing: 0;
  color: #ffffffdf;
}
.gem-back {
  transform: translateZ(-30px);
  background: var(--sculpture-core);
  box-shadow: 0 30px 70px
    color-mix(in srgb, var(--sculpture-purple) 35%, transparent);
}
.gem-side {
  position: absolute;
  top: 8px;
  bottom: 8px;
  right: -30px;
  width: 60px;
  background: linear-gradient(var(--sculpture-core), var(--sculpture-purple));
  transform: rotateY(90deg);
  border-radius: 20px;
}
.gem-shine {
  position: absolute;
  inset: 0;
  border-radius: 38px;
  transform: translateZ(31px);
  background: linear-gradient(
    120deg,
    transparent 20%,
    #ffffff33 45%,
    transparent 65%
  );
  opacity: 0.55;
  pointer-events: none;
}
.sculpture-ring {
  position: absolute;
  inset: -42px;
  border: 1px solid color-mix(in srgb, var(--sculpture-blue) 60%, transparent);
  border-radius: 50%;
  transform-style: preserve-3d;
  pointer-events: none;
}
.ring-a {
  transform: rotateX(65deg) rotateZ(-25deg);
  box-shadow: 0 0 25px
    color-mix(in srgb, var(--sculpture-blue) 12%, transparent);
  animation: ring-orbit-a 18s linear infinite;
}
.ring-b {
  inset: -70px;
  transform: rotateY(65deg) rotateZ(35deg);
  border-color: color-mix(in srgb, var(--sculpture-purple) 45%, transparent);
  animation: ring-orbit-b 23s linear infinite;
}
.ring-c {
  inset: -92px;
  transform: rotateX(65deg) rotateY(-20deg);
  opacity: 0.3;
}
.satellite {
  position: absolute;
  border-radius: 50%;
  background: radial-gradient(
    circle at 28% 25%,
    #ffffffdd,
    var(--sculpture-blue) 25%,
    var(--sculpture-core) 70%
  );
  box-shadow:
    inset -5px -8px 15px #00000025,
    0 10px 25px var(--fas-shadow);
}
.satellite-a {
  width: 45px;
  height: 45px;
  left: -50px;
  top: 10px;
  animation: satellite-drift 7s ease-in-out infinite;
}
.satellite-b {
  width: 27px;
  height: 27px;
  right: -65px;
  bottom: 30px;
  animation: satellite-drift 9s ease-in-out -3s infinite;
}
.satellite-c {
  width: 15px;
  height: 15px;
  right: 5px;
  top: -70px;
  background: var(--sculpture-purple);
}
.sculpture-shadow {
  position: absolute;
  bottom: 50px;
  left: 24%;
  right: 24%;
  height: 45px;
  border-radius: 50%;
  background: var(--sculpture-core);
  opacity: 0.16;
  filter: blur(25px);
  animation: shadow-breathe 8s ease-in-out infinite;
}
.sculpture-label {
  position: absolute;
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 12px 16px;
  border: 1px solid var(--fas-border);
  border-radius: 15px;
  background: color-mix(in srgb, var(--fas-card) 85%, transparent);
  backdrop-filter: blur(12px);
  color: var(--fas-text-soft);
  font-size: 11px;
  box-shadow: 0 14px 35px var(--fas-shadow);
  transition:
    transform 0.7s,
    border-color 0.5s,
    box-shadow 0.5s;
  pointer-events: none;
}
.sculpture-label svg {
  width: 19px;
  height: 19px;
  color: var(--fas-accent-text);
}
.label-0 {
  top: 52px;
  left: 12%;
  transform: rotate(-7deg);
}
.label-1 {
  top: 125px;
  right: 0;
  transform: rotate(7deg);
}
.label-2 {
  bottom: 105px;
  left: 0;
  transform: rotate(-5deg);
}
.label-3 {
  bottom: 42px;
  right: 12%;
  transform: rotate(6deg);
}
.sculpture-label.selected {
  border-color: var(--fas-accent-border);
  box-shadow: 0 14px 40px
    color-mix(in srgb, var(--sculpture-blue) 18%, transparent);
  transform: translateY(-7px) rotate(0deg);
}
.sculpture-caption {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 10px;
  border-top: 1px solid var(--fas-border);
}
.caption-line {
  width: 26px;
  height: 1px;
  background: var(--fas-accent);
}
.sculpture-caption p {
  margin: 0;
}
.sculpture-caption strong {
  display: block;
  color: var(--fas-text);
  font-size: 12px;
  font-weight: 600;
}
.sculpture-caption p > span {
  display: block;
  font-size: 9px;
  margin-top: 5px;
  color: var(--fas-muted);
}
.scene-control-hint {
  margin-left: auto;
  font-size: 9px;
  color: var(--fas-muted);
}
.scene-control-hint > span {
  margin-left: 6px;
}
.sculpture-stage.pressed .fas-gem {
  transform: rotateX(-9deg) rotateY(calc(-25deg + var(--spin, 0deg)))
    rotateZ(2deg) scale(1.06);
}
.sculpture-stage.pressed .sculpture-label {
  transform: translateY(-12px);
}
.platform-grid .feature-card {
  display: block;
  text-align: left;
  cursor: pointer;
  color: var(--fas-text);
  font-family: inherit;
  transition:
    transform 0.5s cubic-bezier(0.2, 0.8, 0.2, 1),
    box-shadow 0.5s,
    border-color 0.5s;
}
.platform-grid .feature-card:hover {
  transform: perspective(900px) rotateX(3deg) translateY(-7px);
  box-shadow: 0 25px 55px var(--fas-shadow);
}
.platform-grid .feature-card:active {
  transform: scale(0.98);
}
.platform-grid .feature-card.chosen {
  border-color: var(--fas-accent-border);
}
.platform-grid .feature-card small > span {
  float: right;
  transition: transform 0.3s;
}
.platform-grid .feature-card:hover small > span {
  transform: translateX(5px);
}
@keyframes sculpture-hover {
  0%,
  100% {
    transform: translateY(0) rotateZ(-2deg);
  }
  50% {
    transform: translateY(-18px) rotateZ(3deg);
  }
}
@keyframes ring-orbit-a {
  from {
    transform: rotateX(65deg) rotateZ(0deg);
  }
  to {
    transform: rotateX(65deg) rotateZ(360deg);
  }
}
@keyframes ring-orbit-b {
  from {
    transform: rotateY(65deg) rotateZ(0deg);
  }
  to {
    transform: rotateY(65deg) rotateZ(-360deg);
  }
}
@keyframes satellite-drift {
  0%,
  100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(20px);
  }
}
@keyframes shadow-breathe {
  0%,
  100% {
    transform: scale(0.9);
    opacity: 0.14;
  }
  50% {
    transform: scale(1.1);
    opacity: 0.1;
  }
}
@media (max-width: 900px) {
  .hero-sculpture {
    width: 100%;
    max-width: 550px;
    justify-self: center;
  }
  .sculpture-stage {
    height: 470px;
  }
  .landing-nav nav {
    display: none;
  }
}
@media (max-width: 600px) {
  .nav-actions {
    gap: 8px;
  }
  .theme-switch {
    width: 37px;
    height: 37px;
  }
  .landing-brand small {
    max-width: 110px;
    line-height: 1.5;
  }
  .landing-enter {
    padding: 10px 12px;
    gap: 7px;
  }
  .sculpture-stage {
    height: 395px;
  }
  .sculpture-float {
    width: 190px;
    height: 190px;
    margin: -95px;
  }
  .gem-front {
    font-size: 95px;
    border-radius: 30px;
  }
  .gem-back,
  .gem-shine {
    border-radius: 30px;
  }
  .gem-front > span {
    font-size: 30px;
  }
  .sculpture-ring {
    inset: -22px;
  }
  .ring-b {
    inset: -40px;
  }
  .ring-c {
    inset: -55px;
  }
  .sculpture-label {
    font-size: 10px;
    padding: 10px 11px;
  }
  .label-0 {
    top: 34px;
    left: 5%;
  }
  .label-1 {
    top: 100px;
    right: 0;
  }
  .label-2 {
    bottom: 95px;
    left: 0;
  }
  .label-3 {
    bottom: 25px;
    right: 5%;
  }
  .scene-control-hint {
    max-width: 90px;
    text-align: right;
    line-height: 1.6;
  }
  .hero-copy h1 {
    font-size: clamp(34px, 8.7vw, 48px);
  }
}
@media (prefers-reduced-motion: reduce) {
  .sculpture-stage,
  .fas-gem {
    transform: none !important;
  }
  .sculpture-stage:active {
    transform: none !important;
  }
   .fas-landing {
    animation: none !important;
    transition: none !important;
  }
}

/* ================================================================
   FAS CINEMATIC SCROLL V3 — GSAP / ScrollTrigger
   CSS bagian ini MENGGANTIKAN engine scroll-chapter lama secara penuh.
   Hanya GSAP yang mengatur transform dan opacity konten antarscene.
   ================================================================ */

.fas-landing {
  overflow-x: clip;
  overflow-y: visible;
  min-height: 100svh;
  /* Lenis controls smoothing, while the browser keeps native scrolling. */
  scroll-behavior: auto;
}

.fas-landing .landing-nav {
  position: relative;
  z-index: 50;
}

.fas-landing .landing-hero {
  min-height: min(800px, calc(100svh - 110px));
  padding-top: clamp(45px, 7vh, 100px);
  padding-bottom: clamp(60px, 8vh, 95px);
  transform-style: preserve-3d;
}

.fas-landing .hero-copy {
  animation: none; /* GSAP handles initial reveal, avoid CSS transform clash */
}

.fas-landing .landing-hero .hero-copy,
.fas-landing .landing-hero .hero-sculpture {
  position: relative;
  will-change: transform, opacity;
}

.fas-landing .scroll-chapter {
  position: relative;
  width: 100%;
  max-width: none;
  min-height: 100svh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: clamp(95px, 12vh, 135px) 0;
  margin: 0;
  box-sizing: border-box;
  isolation: isolate;
  scroll-margin-top: 0;
}

.fas-landing .scroll-chapter::before {
  position: absolute;
  inset: 7% 0;
  z-index: -1;
  content: "";
  pointer-events: none;
  background: radial-gradient(ellipse at 54% 51%, var(--fas-accent-soft), transparent 69%);
  opacity: .8;
}

.fas-landing .scene-stage {
  position: relative;
  width: min(1240px, calc(100% - 96px));
  max-width: 1240px;
  min-width: 0;
  margin: 0 auto;
  padding: 0;
  top: auto;
  perspective: 1300px;
  transform-style: preserve-3d;
}

.fas-landing .platform-section {
  border-top: 1px solid var(--fas-border);
}

.fas-landing .platform-stage > header {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px 38px;
  align-items: end;
}

.fas-landing .platform-stage > header .section-kicker {
  grid-column: 1 / -1;
}

.fas-landing .platform-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 18px;
  margin-top: clamp(26px, 4vh, 40px);
  perspective: 1450px;
}

.fas-landing .platform-grid .feature-card {
  min-width: 0;
  backface-visibility: hidden;
  transform-style: preserve-3d;
  box-shadow: 0 20px 50px var(--fas-shadow-soft);
  transition: border-color .3s ease, box-shadow .35s ease;
  will-change: transform, opacity, filter;
}

.fas-landing .platform-grid .feature-card:hover {
  border-color: var(--fas-accent);
  box-shadow: 0 26px 58px var(--fas-shadow);
}

.fas-landing .workflow-stage,
.fas-landing .company-stage {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: clamp(30px, 6vw, 95px);
  align-items: center;
}

.fas-landing .workflow-stage > div:first-child > *,
.fas-landing .workflow-stage li,
.fas-landing .company-visual,
.fas-landing .company-copy > * {
  will-change: transform, opacity;
  backface-visibility: hidden;
}

.fas-landing .workflow-stage ol,
.fas-landing .workflow-stage li {
  min-width: 0;
}

.fas-landing .workflow-stage li {
  padding: clamp(20px, 3vh, 30px) 0;
}

.fas-landing .company-visual {
  min-height: 360px;
  transform-style: preserve-3d;
  display: grid;
  place-items: center;
}

.fas-landing .company-stage .company-copy .hero-primary {
  width: fit-content;
}

/* Final harus CENTER baik secara horizontal maupun vertikal. */
.fas-landing .finale-section {
  min-height: 92svh;
  background:
    radial-gradient(circle at 50% 45%, var(--fas-accent-soft), transparent 58%),
    transparent;
  text-align: center;
}

.fas-landing .finale-stage {
  width: min(1050px, calc(100% - 56px));
  min-height: min(560px, 70svh);
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 20px;
  text-align: center;
  isolation: isolate;
}

.finale-halo {
  position: absolute;
  left: 50%;
  top: 50%;
  width: min(740px, 95vw);
  aspect-ratio: 1;
  border: 1px solid var(--fas-accent-border);
  border-radius: 50%;
  transform: translate(-50%, -50%);
  z-index: -1;
  opacity: .28;
  pointer-events: none;
}

.finale-halo.halo-two {
  width: min(490px, 70vw);
  opacity: .24;
}

.finale-kicker {
  font-size: 10px;
  letter-spacing: .21em;
  font-weight: 800;
  color: var(--fas-accent-text);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin: 0;
  text-align: center;
}

.finale-kicker-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--fas-accent);
  box-shadow: 0 0 20px var(--fas-accent);
}

.finale-heading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  margin: 4px 0 0;
  font-size: clamp(42px, 6.7vw, 92px);
  line-height: 1.07;
  font-weight: 750;
  letter-spacing: -.068em;
  color: var(--fas-text);
  text-wrap: balance;
}

.finale-line {
  display: block;
  white-space: normal;
  will-change: transform, opacity;
}

.finale-accent {
  color: var(--fas-accent-text);
}

.finale-description {
  max-width: 560px;
  margin: 0 auto;
  font-size: clamp(12px, 1.2vw, 15px);
  line-height: 1.9;
  color: var(--fas-muted);
  text-align: center;
}

.fas-landing .finale-action {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 35px;
  width: fit-content;
  margin: 10px auto 0;
  will-change: transform, opacity;
  transition: background .25s ease, box-shadow .35s ease;
}

.fas-landing .landing-footer {
  position: relative;
  z-index: 1;
  width: min(1240px, calc(100% - 96px));
  margin: 0 auto;
}

/* Bukan fake snap scroll; semua section punya tinggi konten nyata. */
@media (max-width: 1199px), (max-height: 759px) {
  .fas-landing .scroll-chapter {
    min-height: 85svh;
    padding: 100px 0 95px;
  }
  .fas-landing .platform-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .fas-landing .scene-stage {
    width: min(920px, calc(100% - 64px));
  }
  .fas-landing .finale-section {
    min-height: 83svh;
  }
}

@media (max-width: 760px) {
  .fas-landing .landing-nav,
  .fas-landing .landing-hero,
  .fas-landing .landing-footer { width: calc(100% - 36px); }
  .fas-landing .landing-nav { min-height: 83px; }
  .fas-landing .landing-hero {
    min-height: 0;
    padding: 60px 0 75px;
  }
  .fas-landing .scroll-chapter { padding: 85px 0 90px; }
  .fas-landing .scene-stage { width: calc(100% - 36px); }
  .fas-landing .platform-stage > header,
  .fas-landing .workflow-stage,
  .fas-landing .company-stage { grid-template-columns: 1fr; }
  .fas-landing .platform-grid { grid-template-columns: 1fr; }
  .fas-landing .platform-grid .feature-card { min-height: 0; }
  .fas-landing .company-visual { min-height: 280px; }
  .fas-landing .finale-stage { min-height: 490px; gap: 16px; }
  .fas-landing .finale-heading { font-size: clamp(36px, 10vw, 55px); }
  .finale-kicker { font-size: 8px; letter-spacing: .1em; }
  .fas-landing .landing-footer { flex-wrap: wrap; gap: 16px; }
}

@media (prefers-reduced-motion: reduce) {
  .fas-landing .scene-stage *,
  .fas-landing .hero-copy *,
  .fas-landing .hero-sculpture {
    animation: none !important;
    transition: none !important;
    filter: none !important;
  }
}
</style>
