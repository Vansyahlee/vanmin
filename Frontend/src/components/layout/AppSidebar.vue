<script setup>
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const props = defineProps({
  collapsed: { type: Boolean, default: false },
})

const emit = defineEmits(['update:collapsed'])
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const user = computed(() => authStore.user)
const profileOpen = ref(false)
const financeOpen = ref(route.path.startsWith('/finance'))

const displayRole = computed(() => ({
  admin: 'Administrator',
  pegawai: 'Pegawai Gudang',
  finance: 'Finance & Accounting',
}[user.value?.role] || 'Pengguna'))

const navItems = [
  { name: 'Overview', path: '/dashboard', roles: ['admin'], icon: 'M3 13h8V3H3v10Zm10 8h8V11h-8v10ZM3 21h8v-6H3v6Zm10-12h8V3h-8v6Z' },
  { name: 'Dashboard Gudang', path: '/warehouse', roles: ['pegawai'], icon: 'M3 9l9-6 9 6v12H3V9Zm4 4h4v4H7v-4Zm7 0h4v4h-4v-4Z' },
  { name: 'Operations', path: '/operations', roles: ['admin', 'pegawai'], icon: 'M3 6h18M6 6V4h12v2M5 6l1 14h12l1-14M9 10v6M15 10v6' },
  { name: 'Inventory', path: '/inventory', roles: ['admin', 'pegawai'], icon: 'm21 8-9-5-9 5 9 5 9-5ZM3 8v8l9 5 9-5V8M12 13v8' },
  { name: 'Distribution', path: '/distribution', roles: ['admin', 'pegawai'], icon: 'M3 17h2M7 17h10M19 17h2M5 17V7h10v10M15 11h3l3 3v3M8 10h4M8 13h4' },
  { name: 'Intelligence', path: '/intelligence', roles: ['admin', 'pegawai', 'finance'], icon: 'M9 18h6M10 22h4M8 14c-2-1.3-3-3.2-3-5a7 7 0 0 1 14 0c0 1.8-1 3.7-3 5-1 .7-1 1.3-1 2H9c0-.7 0-1.3-1-2Z' },
  { name: 'Finance', path: '/finance', roles: ['admin', 'finance'], icon: 'M3 10h18M5 10v8M9 10v8M15 10v8M19 10v8M3 18h18M12 3 3 8h18l-9-5Z', expandable: true },
  { name: 'Reports', path: '/reports', roles: ['admin', 'finance'], icon: 'M5 3h10l4 4v14H5V3Zm10 0v5h5M9 13h6M9 17h6M9 9h2' },
  { name: 'Users', path: '/users', roles: ['admin'], icon: 'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75' },
]

const financePages = [
  { name: 'Overview', path: '/finance' },
  { name: 'Analytics', path: '/finance/analytics' },
  { name: 'Penjualan', path: '/finance/sales' },
  { name: 'Customer', path: '/finance/customers' },
  { name: 'Piutang', path: '/finance/receivables' },
  { name: 'Pembayaran', path: '/finance/payments' },
  { name: 'Pengeluaran', path: '/finance/expenses' },
  { name: 'Kas & Bank', path: '/finance/cash' },
  { name: 'Laba', path: '/finance/profit' },
  { name: 'Pajak', path: '/finance/tax' },
  { name: 'Laporan', path: '/finance/reports' },
  { name: 'Notifikasi', path: '/finance/notifications' },
]

const visibleNavItems = computed(() => navItems.filter(
  item => item.roles.includes(user.value?.role),
))

const isActive = (item) => item.expandable
  ? route.path === '/finance' || route.path.startsWith('/finance/')
  : route.path === item.path

watch(() => route.path, (path) => {
  profileOpen.value = false
  if (path === '/finance' || path.startsWith('/finance/')) {
    financeOpen.value = true
  }
})

const toggleSidebar = () => {
  profileOpen.value = false
  emit('update:collapsed', !props.collapsed)
}

const toggleFinance = () => {
  profileOpen.value = false
  if (props.collapsed) {
    emit('update:collapsed', false)
    financeOpen.value = true
  } else {
    financeOpen.value = !financeOpen.value
  }
}

const toggleProfile = () => {
  if (props.collapsed) {
    emit('update:collapsed', false)
    profileOpen.value = true
    return
  }
  profileOpen.value = !profileOpen.value
}

const goProfile = async () => {
  profileOpen.value = false
  await router.push('/profile')
}

const goSettings = async () => {
  profileOpen.value = false
  await router.push('/settings')
}

const logout = async () => {
  profileOpen.value = false
  authStore.logout()
  await router.push('/login')
}
</script>

<template>
  <aside
    class="fas-app-sidebar"
    :class="{ 'is-collapsed': collapsed }"
  >
    <div class="fas-sidebar-brand">
      <div class="fas-sidebar-brand-main">
        <div class="fas-sidebar-monogram">F</div>
        <div v-if="!collapsed" class="fas-sidebar-brand-copy">
          <strong>PT. FAS</strong>
          <span>Business Platform</span>
        </div>
      </div>
      <button
        v-if="!collapsed"
        class="fas-sidebar-toggle"
        type="button"
        aria-label="Sembunyikan sidebar"
        title="Sembunyikan sidebar"
        @click="toggleSidebar"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9"><path d="m15 18-6-6 6-6" /></svg>
      </button>
    </div>

    <div v-if="collapsed" class="fas-sidebar-expand">
      <button
        class="fas-sidebar-toggle"
        type="button"
        aria-label="Tampilkan sidebar"
        title="Tampilkan sidebar"
        @click="toggleSidebar"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9"><path d="m9 18 6-6-6-6" /></svg>
      </button>
    </div>

    <nav class="fas-sidebar-nav" aria-label="Navigasi PT. FAS">
      <template v-for="item in visibleNavItems" :key="item.path">
        <div v-if="item.expandable" class="fas-sidebar-group">
          <button
            type="button"
            class="fas-sidebar-item fas-sidebar-finance-trigger"
            :class="{ 'is-active': isActive(item), 'is-collapsed-item': collapsed }"
            :aria-expanded="!collapsed && financeOpen"
            aria-controls="fas-sidebar-finance-submenu"
            :aria-label="collapsed ? 'Buka submenu Finance' : undefined"
            :title="collapsed ? 'Buka submenu Finance' : undefined"
            @click="toggleFinance"
          >
            <span v-if="isActive(item)" class="fas-sidebar-active-bar"></span>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
              <path :d="item.icon" />
            </svg>
            <span v-if="!collapsed" class="fas-sidebar-label">Finance</span>
            <svg
              v-if="!collapsed"
              class="fas-sidebar-chevron"
              :class="{ 'is-rotated': financeOpen }"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              aria-hidden="true"
            >
              <path d="m6 9 6 6 6-6" />
            </svg>
          </button>
          <div
            v-show="!collapsed && financeOpen"
            id="fas-sidebar-finance-submenu"
            class="fas-sidebar-subnav"
          >
            <RouterLink
              v-for="page in financePages"
              :key="page.path"
              :to="page.path"
              class="fas-sidebar-subitem"
              :class="{ 'is-active': route.path === page.path }"
              :aria-current="route.path === page.path ? 'page' : undefined"
              @click="profileOpen = false"
            >
              <span class="fas-sidebar-subdot"></span>
              <span>{{ page.name }}</span>
            </RouterLink>
          </div>
        </div>

        <RouterLink
          v-else
          :to="item.path"
          class="fas-sidebar-item"
          :class="{ 'is-active': isActive(item), 'is-collapsed-item': collapsed }"
          :title="collapsed ? item.name : undefined"
          :aria-label="collapsed ? item.name : undefined"
          :aria-current="route.path === item.path ? 'page' : undefined"
          @click="profileOpen = false"
        >
          <span v-if="isActive(item)" class="fas-sidebar-active-bar"></span>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path :d="item.icon" /></svg>
          <span v-if="!collapsed" class="fas-sidebar-label">{{ item.name }}</span>
        </RouterLink>
      </template>
    </nav>

    <div class="fas-sidebar-account">
      <div v-if="profileOpen && !collapsed" class="fas-sidebar-account-menu">
        <div class="fas-sidebar-account-identity">
          <strong>{{ user?.name || 'Pengguna PT. FAS' }}</strong>
          <span>{{ user?.email }}</span>
        </div>
        <button type="button" @click="goProfile">Profil Saya</button>
        <button type="button" @click="goSettings">Pengaturan</button>
        <button type="button" class="fas-sidebar-logout" @click="logout">Keluar</button>
      </div>
      <button type="button" class="fas-sidebar-account-trigger" :title="collapsed ? displayRole : undefined" @click="toggleProfile">
        <span class="fas-sidebar-user-avatar">{{ user?.name?.charAt(0)?.toUpperCase() || 'U' }}</span>
        <span v-if="!collapsed" class="fas-sidebar-user-meta"><strong>{{ user?.name || 'Pengguna PT. FAS' }}</strong><small>{{ displayRole }}</small></span>
        <svg v-if="!collapsed" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" :class="{ 'is-rotated': profileOpen }"><path d="m6 15 6-6 6 6" /></svg>
      </button>
    </div>
  </aside>
</template>

<style scoped>
.fas-app-sidebar { position:fixed; inset:0 auto 0 0; z-index:50; width:280px; display:flex; flex-direction:column; min-height:0; border-right:1px solid var(--fas-border); background:var(--fas-surface); color:var(--fas-text); box-shadow:var(--fas-shadow-soft); transition:width .25s ease; }
.fas-app-sidebar.is-collapsed { width:88px; }
.fas-sidebar-brand { display:flex; align-items:center; justify-content:space-between; flex-shrink:0; min-height:96px; padding:0 19px; border-bottom:1px solid var(--fas-border); gap:8px; }
.fas-sidebar-brand-main { display:flex; align-items:center; gap:12px; min-width:0; }
.fas-sidebar-monogram { flex:0 0 46px; height:46px; border-radius:16px; display:grid; place-items:center; background:linear-gradient(145deg,#3186ff,#1553e4); color:#fff; font-size:21px; font-weight:850; }
.fas-sidebar-brand-copy { display:flex; flex-direction:column; gap:3px; min-width:0; }
.fas-sidebar-brand-copy strong { font-size:15px; font-weight:850; white-space:nowrap; }
.fas-sidebar-brand-copy span { color:var(--fas-muted); font-size:10px; letter-spacing:.15em; text-transform:uppercase; white-space:nowrap; }
.fas-sidebar-toggle { display:grid; place-items:center; flex:0 0 36px; width:36px; height:36px; border:1px solid var(--fas-border); border-radius:12px; color:var(--fas-muted); background:var(--fas-elevated); cursor:pointer; }
.fas-sidebar-toggle svg { width:16px; height:16px; }
.fas-sidebar-expand { display:grid; place-items:center; padding:14px 0 4px; }
.fas-sidebar-nav { flex:1; min-height:0; overflow-x:hidden; overflow-y:auto; padding:18px 14px 22px; display:flex; flex-direction:column; gap:5px; scrollbar-width:thin; scrollbar-color:var(--fas-border-strong) transparent; }
.fas-sidebar-item { position:relative; display:flex; align-items:center; gap:13px; flex-shrink:0; min-height:46px; width:100%; padding:10px 13px; text-decoration:none; border:0; border-radius:15px; background:transparent; color:var(--fas-muted); text-align:left; font-size:13px; font-weight:650; cursor:pointer; transition:background .16s ease,color .16s ease; }
.fas-sidebar-item > svg { flex:0 0 20px; width:20px; height:20px; }
.fas-sidebar-item:hover, .fas-sidebar-item.is-active { background:var(--fas-accent-soft); color:var(--fas-accent-text); }
.fas-sidebar-item.is-active { font-weight:750; }
.fas-sidebar-active-bar { position:absolute; inset:12px auto 12px 0; width:4px; border-radius:0 5px 5px 0; background:var(--fas-accent); }
.fas-sidebar-label { flex:1; white-space:nowrap; }
.fas-sidebar-item.is-collapsed-item { justify-content:center; padding:10px; }
.fas-sidebar-finance-trigger { justify-content:flex-start; }
.fas-sidebar-finance-trigger.is-collapsed-item { justify-content:center; }
.fas-sidebar-chevron { width:16px !important; height:16px !important; flex-shrink:0 !important; margin-left:auto; transition:transform .18s ease; }
.fas-sidebar-chevron.is-rotated { transform:rotate(180deg); }
.fas-sidebar-subnav { margin:3px 0 8px 21px; padding:4px 0 4px 13px; border-left:1px solid var(--fas-border-strong); display:flex; flex-direction:column; gap:2px; }
.fas-sidebar-subitem { display:flex; align-items:center; gap:10px; min-height:34px; border-radius:9px; padding:6px 11px; color:var(--fas-muted); font-size:12px; font-weight:550; text-decoration:none; transition:background .16s ease,color .16s ease; }
.fas-sidebar-subitem:hover { background:var(--fas-hover); color:var(--fas-text); }
.fas-sidebar-subitem.is-active { background:var(--fas-accent-soft); color:var(--fas-accent-text); font-weight:750; }
.fas-sidebar-subdot { flex:0 0 5px; width:5px; height:5px; border-radius:50%; background:currentColor; opacity:.55; }
.fas-sidebar-subitem.is-active .fas-sidebar-subdot { opacity:1; }
.fas-sidebar-account { position:relative; padding:13px 14px; flex-shrink:0; border-top:1px solid var(--fas-border); }
.fas-sidebar-account-trigger { display:flex; width:100%; min-width:0; align-items:center; gap:11px; padding:9px; border:1px solid var(--fas-border); border-radius:16px; background:var(--fas-elevated); color:var(--fas-text); text-align:left; cursor:pointer; }
.fas-sidebar-user-avatar { width:39px; height:39px; display:grid; place-items:center; flex-shrink:0; border-radius:12px; background:linear-gradient(140deg,#334155,#101827); color:white; font-size:14px; font-weight:800; }
.fas-sidebar-user-meta { display:flex; flex:1; min-width:0; flex-direction:column; gap:3px; }
.fas-sidebar-user-meta strong { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; font-size:12px; }
.fas-sidebar-user-meta small { color:var(--fas-muted); font-size:11px; }
.fas-sidebar-account-trigger > svg { width:16px; height:16px; color:var(--fas-muted); transition:transform .18s; }
.fas-sidebar-account-trigger > svg.is-rotated { transform:rotate(180deg); }
.fas-sidebar-account-menu { position:absolute; left:14px; right:14px; bottom:calc(100% + 5px); padding:9px; background:var(--fas-card); border:1px solid var(--fas-border-strong); box-shadow:var(--fas-shadow-card); border-radius:14px; z-index:5; }
.fas-sidebar-account-identity { display:flex; flex-direction:column; gap:5px; padding:10px; border-bottom:1px solid var(--fas-border); margin-bottom:6px; }
.fas-sidebar-account-identity strong { font-size:12px; }
.fas-sidebar-account-identity span { font-size:11px; overflow-wrap:anywhere; color:var(--fas-muted); }
.fas-sidebar-account-menu button { display:block; width:100%; border:0; background:transparent; color:var(--fas-text-soft); text-align:left; padding:10px 11px; border-radius:9px; font-size:12px; cursor:pointer; }
.fas-sidebar-account-menu button:hover { background:var(--fas-hover); }
.fas-sidebar-account-menu .fas-sidebar-logout { color:var(--fas-danger); }
.fas-app-sidebar :is(a,button):focus-visible { outline:2px solid var(--fas-accent); outline-offset:2px; }
@media (max-width:1023px) { .fas-app-sidebar { box-shadow:0 15px 45px rgba(0,0,0,.18); } }
@media (prefers-reduced-motion:reduce) { .fas-app-sidebar *, .fas-app-sidebar { transition:none !important; } }
</style>
