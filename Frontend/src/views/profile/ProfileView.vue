<script setup>
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()

const user = computed(() => authStore.user)

const displayRole = computed(() => {
  const roles = {
    admin: 'Administrator',
    pegawai: 'Warehouse Operations',
    finance: 'Finance & Accounting',
  }

  return roles[user.value?.role] || 'User'
})
</script>

<template>
  <div class="p-8 xl:p-10">
    <div class="max-w-4xl">
      <p class="text-sm font-medium text-blue-400">
        Account
      </p>

      <h2 class="mt-2 text-3xl font-bold text-white">
        My Profile
      </h2>

      <p class="mt-3 text-sm text-slate-500">
        Informasi akun internal PT. FAS.
      </p>

      <div
        class="mt-8 rounded-3xl border border-white/[0.06] bg-[#0c121e]/80 p-7"
      >
        <div class="flex items-center gap-5">
          <div
            class="flex h-20 w-20 items-center justify-center rounded-2xl bg-gradient-to-br from-blue-500 to-blue-700 text-2xl font-bold text-white"
          >
            {{ user?.name?.charAt(0) || 'U' }}
          </div>

          <div>
            <h3 class="text-xl font-semibold text-white">
              {{ user?.name }}
            </h3>

            <p class="mt-1 text-sm text-slate-500">
              {{ displayRole }}
            </p>
          </div>
        </div>

        <div
          class="mt-8 grid gap-4 border-t border-white/5 pt-6 md:grid-cols-2"
        >
          <div class="rounded-2xl bg-white/[0.025] p-4">
            <p class="text-xs uppercase tracking-wider text-slate-600">
              Name
            </p>

            <p class="mt-2 text-sm font-medium text-slate-200">
              {{ user?.name }}
            </p>
          </div>

          <div class="rounded-2xl bg-white/[0.025] p-4">
            <p class="text-xs uppercase tracking-wider text-slate-600">
              Email
            </p>

            <p class="mt-2 text-sm font-medium text-slate-200">
              {{ user?.email }}
            </p>
          </div>

          <div class="rounded-2xl bg-white/[0.025] p-4">
            <p class="text-xs uppercase tracking-wider text-slate-600">
              Role
            </p>

            <p class="mt-2 text-sm font-medium text-slate-200">
              {{ displayRole }}
            </p>
          </div>

          <div class="rounded-2xl bg-white/[0.025] p-4">
            <p class="text-xs uppercase tracking-wider text-slate-600">
              Account Status
            </p>

            <p class="mt-2 text-sm font-medium text-emerald-400">
              Active
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>