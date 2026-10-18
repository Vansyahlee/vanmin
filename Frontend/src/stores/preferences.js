import { defineStore } from 'pinia'

export const usePreferencesStore = defineStore('preferences', {
    state: () => ({
        appearance: localStorage.getItem('fas_appearance') || 'dark',
        accent: localStorage.getItem('fas_accent') || 'blue',
        sidebarMode:
            localStorage.getItem('fas_sidebar_mode') || 'expanded',
        density:
            localStorage.getItem('fas_density') || 'comfortable',
        brightness: Number(
            localStorage.getItem('fas_brightness') || 100,
        ),
        systemListenerReady: false,
    }),

    actions: {
        initialize() {
            this.applyAppearance()
            this.applyAccent()
            this.applyDensity()
            this.applyBrightness()

            if (!this.systemListenerReady) {
                const media = window.matchMedia(
                    '(prefers-color-scheme: dark)',
                )

                media.addEventListener('change', () => {
                    if (this.appearance === 'system') {
                        this.applyAppearance()
                    }
                })

                this.systemListenerReady = true
            }
        },

        setAppearance(value) {
            this.appearance = value
            localStorage.setItem('fas_appearance', value)
            this.applyAppearance()
        },

        setAccent(value) {
            this.accent = value
            localStorage.setItem('fas_accent', value)
            this.applyAccent()
        },

        setSidebarMode(value) {
            this.sidebarMode = value

            localStorage.setItem(
                'fas_sidebar_mode',
                value,
            )

            localStorage.setItem(
                'fas_sidebar_collapsed',
                value === 'collapsed' ? 'true' : 'false',
            )
        },

        setDensity(value) {
            this.density = value
            localStorage.setItem('fas_density', value)
            this.applyDensity()
        },

        setBrightness(value) {
            const brightness = Math.min(
                115,
                Math.max(80, Number(value)),
            )

            this.brightness = brightness

            localStorage.setItem(
                'fas_brightness',
                String(brightness),
            )

            this.applyBrightness()
        },

        resetBrightness() {
            this.setBrightness(100)
        },

        applyAppearance() {
            const root = document.documentElement

            let darkMode = this.appearance === 'dark'

            if (this.appearance === 'system') {
                darkMode = window.matchMedia(
                    '(prefers-color-scheme: dark)',
                ).matches
            }

            root.dataset.theme = darkMode
                ? 'dark'
                : 'light'
        },

        applyAccent() {
            document.documentElement.dataset.accent =
                this.accent
        },

        applyDensity() {
            document.documentElement.dataset.density =
                this.density
        },

        applyBrightness() {
            const root = document.documentElement

            if (this.brightness < 100) {
                const opacity =
                    ((100 - this.brightness) / 20) * 0.16

                root.style.setProperty(
                    '--fas-brightness-color',
                    '#000000',
                )

                root.style.setProperty(
                    '--fas-brightness-opacity',
                    String(opacity),
                )

                return
            }

            if (this.brightness > 100) {
                const opacity =
                    ((this.brightness - 100) / 15) * 0.08

                root.style.setProperty(
                    '--fas-brightness-color',
                    '#ffffff',
                )

                root.style.setProperty(
                    '--fas-brightness-opacity',
                    String(opacity),
                )

                return
            }

            root.style.setProperty(
                '--fas-brightness-opacity',
                '0',
            )
        },
    },
})