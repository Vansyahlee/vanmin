<script setup>
import { computed, ref, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'

import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

import { usePreferencesStore } from '@/stores/preferences'

const preferences = usePreferencesStore()
const route = useRoute()

const mobileSidebarOpen = ref(false)

const sidebarCollapsed = computed({
  get() {
    return preferences.sidebarMode === 'collapsed'
  },

  set(value) {
    preferences.setSidebarMode(
      value ? 'collapsed' : 'expanded',
    )
  },
})

watch(
  () => route.fullPath,
  () => {
    mobileSidebarOpen.value = false
  },
)

const toggleMobileSidebar = () => {
  mobileSidebarOpen.value =
    !mobileSidebarOpen.value
}
</script>

<template>
  <div
    class="fas-shell min-h-screen"
    style="
      background: var(--fas-bg);
      color: var(--fas-text);
    "
  >
    <div
      class="pointer-events-none fixed inset-0 overflow-hidden"
    >
      <div
        class="absolute -left-24 top-0 h-[420px] w-[420px] rounded-full blur-[120px]"
        style="
          background: color-mix(
            in srgb,
            var(--fas-accent) 10%,
            transparent
          );
        "
      ></div>
    </div>

    <div
      v-if="mobileSidebarOpen"
      class="fixed inset-0 z-40 bg-black/45 backdrop-blur-sm lg:hidden"
      @click="mobileSidebarOpen = false"
    ></div>

    <AppSidebar
      :collapsed="
        mobileSidebarOpen
          ? false
          : sidebarCollapsed
      "
      :class="[
        mobileSidebarOpen
          ? 'translate-x-0'
          : '-translate-x-full',
        'lg:translate-x-0',
      ]"
      @update:collapsed="
        sidebarCollapsed = $event
      "
    />

    <div
      :class="[
        'relative min-h-screen transition-all duration-300',
        'ml-0',
        sidebarCollapsed
          ? 'lg:ml-[88px]'
          : 'lg:ml-[280px]',
      ]"
    >
      <AppHeader
        @toggle-mobile-sidebar="
          toggleMobileSidebar
        "
      />

      <main
        class="relative min-h-[calc(100vh-3.5rem)] sm:min-h-[calc(100vh-4rem)]"
      >
        <RouterView />
      </main>
    </div>
  </div>
</template>