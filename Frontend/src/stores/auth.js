import { defineStore } from 'pinia'
import api from '@/services/api'

export const useAuthStore = defineStore('auth', {
    state: () => ({
        token: localStorage.getItem('fas_token'),
        user: JSON.parse(localStorage.getItem('fas_user') || 'null'),
    }),

    getters: {
        isAuthenticated: (state) => Boolean(state.token),
        role: (state) => state.user?.role || null,
    },

    actions: {
        async login(email, password) {
            const response = await api.post('/auth/login', {
                email,
                password,
            })

            this.token = response.data.token
            this.user = response.data.user

            localStorage.setItem('fas_token', this.token)
            localStorage.setItem('fas_user', JSON.stringify(this.user))

            return this.user
        },

        logout() {
            this.token = null
            this.user = null

            localStorage.removeItem('fas_token')
            localStorage.removeItem('fas_user')
        },
    },
})