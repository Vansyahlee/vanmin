<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
const router = useRouter()
import { usePreferencesStore } from '@/stores/preferences'

const preferences = usePreferencesStore()

const appearance = computed(() => preferences.appearance)
const accent = computed(() => preferences.accent)
const sidebarMode = computed(() => preferences.sidebarMode)
const density = computed(() => preferences.density)
const brightness = computed(() => preferences.brightness)
</script>

<template>
  <div class="p-5 sm:p-6 lg:p-8 xl:p-10">
    <div class="mx-auto max-w-5xl">
 <button type="button" class="mb-6 rounded-xl border px-5 py-3 text-sm font-semibold" style="color:var(--fas-accent-text);background:var(--fas-accent-soft);border-color:var(--fas-accent-border)" @click="router.push('/settings/company')">Profil &amp; Alamat Perusahaan</button>
      <p
        class="text-sm font-medium"
        style="color: var(--fas-accent-text)"
      >
        Preferences
      </p>

      <h2
        class="mt-2 text-2xl font-bold sm:text-3xl"
        style="color: var(--fas-text)"
      >
        Account & Preferences
      </h2>

      <p
        class="mt-3 text-sm"
        style="color: var(--fas-muted)"
      >
        Sesuaikan tampilan dan pengalaman penggunaan sistem.
      </p>

      <div class="mt-8 space-y-6">
        <section class="settings-card">
          <h3 class="settings-title">
            Appearance
          </h3>

          <p class="settings-description">
            Pilih mode tampilan dashboard.
          </p>

          <div
            class="mt-5 grid gap-3 sm:grid-cols-3"
          >
            <button
              v-for="option in ['dark', 'light', 'system']"
              :key="option"
              type="button"
              :class="[
                'option-button capitalize',
                appearance === option
                  ? 'option-active'
                  : '',
              ]"
              @click="preferences.setAppearance(option)"
            >
              {{ option }}
            </button>
          </div>
        </section>

        <section class="settings-card">
          <h3 class="settings-title">
            Accent Color
          </h3>

          <p class="settings-description">
            Pilih warna aksen interface.
          </p>

          <div class="mt-5 flex flex-wrap gap-3">
            <button
              v-for="option in [
                'blue',
                'indigo',
                'emerald',
              ]"
              :key="option"
              type="button"
              :class="[
                'option-button capitalize',
                accent === option
                  ? 'option-active'
                  : '',
              ]"
              @click="preferences.setAccent(option)"
            >
              {{ option }}
            </button>
          </div>
        </section>

        <section class="settings-card">
          <h3 class="settings-title">
            Sidebar
          </h3>

          <p class="settings-description">
            Tentukan kondisi sidebar saat aplikasi dibuka.
          </p>

          <div
            class="mt-5 grid gap-3 sm:grid-cols-2"
          >
            <button
              v-for="option in [
                'expanded',
                'collapsed',
              ]"
              :key="option"
              type="button"
              :class="[
                'option-button capitalize',
                sidebarMode === option
                  ? 'option-active'
                  : '',
              ]"
              @click="preferences.setSidebarMode(option)"
            >
              {{ option }}
            </button>
          </div>
        </section>

        <section class="settings-card">
          <h3 class="settings-title">
            Interface Density
          </h3>

          <p class="settings-description">
            Atur tingkat kepadatan tampilan dashboard.
          </p>

          <div
            class="mt-5 grid gap-3 sm:grid-cols-2"
          >
            <button
              v-for="option in [
                'comfortable',
                'compact',
              ]"
              :key="option"
              type="button"
              :class="[
                'option-button capitalize',
                density === option
                  ? 'option-active'
                  : '',
              ]"
              @click="preferences.setDensity(option)"
            >
              {{ option }}
            </button>
          </div>
        </section>

        <section class="settings-card">
          <div
            class="flex flex-col justify-between gap-4 sm:flex-row sm:items-center"
          >
            <div>
              <h3 class="settings-title">
                Brightness
              </h3>

              <p class="settings-description">
                Sesuaikan tingkat terang tampilan dashboard.
              </p>
            </div>

            <div
              class="rounded-xl px-4 py-2 text-sm font-semibold"
              style="
                background: var(--fas-accent-soft);
                color: var(--fas-accent-text);
              "
            >
              {{ brightness }}%
            </div>
          </div>

          <div class="mt-6">
            <input
              :value="brightness"
              type="range"
              min="80"
              max="115"
              step="1"
              class="brightness-slider w-full"
              @input="
                preferences.setBrightness(
                  $event.target.value,
                )
              "
            />

            <div
              class="mt-2 flex justify-between text-xs"
              style="color: var(--fas-muted)"
            >
              <span>Darker</span>
              <span>Normal</span>
              <span>Brighter</span>
            </div>

            <button
              type="button"
              class="mt-5 rounded-xl px-4 py-2 text-xs font-semibold transition"
              style="
                border: 1px solid var(--fas-border);
                color: var(--fas-muted);
              "
              @click="preferences.resetBrightness()"
            >
              Reset 100%
            </button>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-card {
  border: 1px solid var(--fas-border);
  border-radius: 1.5rem;
  background: var(--fas-card);
  padding: 1.5rem;
  box-shadow: 0 20px 55px var(--fas-shadow);
  transition:
    background-color 250ms ease,
    border-color 250ms ease;
}

.settings-title {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--fas-text);
}

.settings-description {
  margin-top: 0.35rem;
  font-size: 0.78rem;
  color: var(--fas-muted);
}

.option-button {
  border: 1px solid var(--fas-border);
  border-radius: 0.9rem;
  background: var(--fas-overlay);
  padding: 0.85rem 1rem;
  font-size: 0.8rem;
  color: var(--fas-muted);
  transition: 200ms ease;
}

.option-button:hover {
  background: var(--fas-hover);
  color: var(--fas-text);
}

.option-active {
  border-color: var(--fas-accent-border);
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}

.brightness-slider {
  accent-color: var(--fas-accent);
  cursor: pointer;
}
</style>