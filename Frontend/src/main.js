import './assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import router from './router'
import { usePreferencesStore } from './stores/preferences'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

const preferencesStore = usePreferencesStore()
preferencesStore.initialize()

app.mount('#app')