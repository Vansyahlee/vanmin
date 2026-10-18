<script setup>
import { computed } from 'vue'

const props = defineProps({
  name: {
    type: String,
    required: true,
  },
})

const paths = {
  search: ['M21 21l-4.5-4.5', 'M10.5 18a7.5 7.5 0 1 0 0-15 7.5 7.5 0 0 0 0 15Z'],
  'chevron-down': ['m6 9 6 6 6-6'],
  close: ['m6 6 12 12', 'M18 6 6 18'],
  refresh: ['M20 7v5h-5', 'M4 17v-5h5', 'M6.1 6.1A8 8 0 0 1 20 12', 'M4 12a8 8 0 0 0 13.9 5.9'],
  'panel-right': ['M4 3h16a1 1 0 0 1 1 1v16a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1Z', 'M15 3v18'],
  check: ['m5 12 4 4L19 6'],
  enter: ['M20 4v8H4', 'm9 7-5 5 5 5'],
  calendar: ['M5 5h14a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Z', 'M16 3v4', 'M8 3v4', 'M3 11h18'],
  plus: ['M12 5v14', 'M5 12h14'],
  minus: ['M5 12h14'],
}

const iconPaths = computed(() => paths[props.name] || paths['chevron-down'])
</script>

<template>
  <svg
    class="fas-icon"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    focusable="false"
  >
    <path v-for="path in iconPaths" :key="path" :d="path" />
  </svg>
</template>

<style scoped>
.fas-icon {
  display: inline-block;
  width: 1.125rem;
  height: 1.125rem;
  flex-shrink: 0;
  vertical-align: middle;
}
</style>
