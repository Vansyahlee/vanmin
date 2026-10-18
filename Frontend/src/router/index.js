
import {
  createRouter,
  createWebHistory,
} from 'vue-router'

import DashboardLayout from '@/components/layout/DashboardLayout.vue'

import LandingView from '@/views/LandingView.vue'
import LoginView from '@/views/auth/LoginView.vue'

import HomePage from '@/views/dashboard/HomePage.vue'
import WarehouseDashboardView from '@/views/warehouse/WarehouseDashboardView.vue'

import OperationsView from '@/views/operations/OperationsView.vue'
import InventoryView from '@/views/inventory/InventoryView.vue'
import DistributionView from '@/views/distribution/DistributionView.vue'
import IntelligenceView from '@/views/intelligence/IntelligenceView.vue'

import FinanceOverviewView from '@/views/finance/FinanceOverviewView.vue'
import FinanceAnalyticsView from '@/views/finance/FinanceAnalyticsView.vue'
import FinanceSalesView from '@/views/finance/FinanceSalesView.vue'
import FinanceCustomersView from '@/views/finance/FinanceCustomersView.vue'
import FinanceReceivablesView from '@/views/finance/FinanceReceivablesView.vue'
import FinancePaymentsView from '@/views/finance/FinancePaymentsView.vue'
import FinanceExpensesView from '@/views/finance/FinanceExpensesView.vue'
import FinanceNotificationsView from '@/views/finance/FinanceNotificationsView.vue'
import FinanceCashView from '@/views/finance/FinanceCashView.vue'
import FinanceProfitView from '@/views/finance/FinanceProfitView.vue'
import FinanceTaxView from '@/views/finance/FinanceTaxView.vue'
import FinanceReportsView from '@/views/finance/FinanceReportsView.vue'

import ReportsView from '@/views/reports/ReportsView.vue'
import UsersView from '@/views/users/UsersView.vue'
import ProfileView from '@/views/profile/ProfileView.vue'

import SettingsView from '@/views/settings/SettingsView.vue'
import CompanySettingsView from '@/views/settings/CompanySettingsView.vue'

export const homeForRole = (role) => {
  if (role === 'pegawai') {
    return '/warehouse'
  }

  if (role === 'finance') {
    return '/finance'
  }

  return '/dashboard'
}

const router = createRouter({
  history: createWebHistory(
    import.meta.env.BASE_URL,
  ),

  routes: [
    {
      path: '/',
      name: 'landing',
      component: LandingView,
    },

    {
      path: '/login',
      name: 'login',
      component: LoginView,
    },

    {
      path: '/',
      component: DashboardLayout,

      meta: {
        requiresAuth: true,
      },

      children: [
        {
          path: 'dashboard',
          name: 'dashboard',
          component: HomePage,

          meta: {
            roles: [
              'admin',
            ],
          },
        },

        {
          path: 'warehouse',
          name: 'warehouse',
          component: WarehouseDashboardView,

          meta: {
            roles: [
              'admin',
              'pegawai',
            ],
          },
        },

        {
          path: 'operations',
          name: 'operations',
          component: OperationsView,

          meta: {
            roles: [
              'admin',
              'pegawai',
            ],
          },
        },

        {
          path: 'inventory',
          name: 'inventory',
          component: InventoryView,

          meta: {
            roles: [
              'admin',
              'pegawai',
            ],
          },
        },

        {
          path: 'distribution',
          name: 'distribution',
          component: DistributionView,

          meta: {
            roles: [
              'admin',
              'pegawai',
            ],
          },
        },

        {
          path: 'intelligence',
          name: 'intelligence',
          component: IntelligenceView,

          meta: {
            roles: [
              'admin',
              'pegawai',
              'finance',
            ],
          },
        },

        {
          path: 'finance',
          name: 'finance',
          component: FinanceOverviewView,

          meta: {
            roles: [
              'admin',
              'finance',
            ],
          },
        },

        {
          path: 'finance/analytics',
          name: 'finance-analytics',
          component: FinanceAnalyticsView,

          meta: {
            roles: [
              'admin',
              'finance',
            ],
          },
        },

        {
          path: 'finance/sales',
          name: 'finance-sales',
          component: FinanceSalesView,

          meta: {
            roles: [
              'admin',
              'finance',
            ],
          },
        },

        {
          path: 'finance/customers',
          name: 'finance-customers',
          component: FinanceCustomersView,

          meta: {
            roles: [
              'admin',
              'finance',
            ],
          },
        },

        {
          path: 'finance/receivables',
          name: 'finance-receivables',
          component: FinanceReceivablesView,

          meta: {
            roles: [
              'admin',
              'finance',
            ],
          },
        },

        {
          path: 'finance/payments',
          name: 'finance-payments',
          component: FinancePaymentsView,

          meta: {
            roles: [
              'admin',
              'finance',
            ],
          },
        },

        {
          path: 'finance/notifications',
          name: 'finance-notifications',
          component: FinanceNotificationsView,

          meta: {
            roles: [
              'admin',
              'finance',
            ],
          },
        },

        {
          path: 'finance/expenses',
          name: 'finance-expenses',
          component: FinanceExpensesView,

          meta: {
            roles: [
              'admin',
              'finance',
            ],
          },
        },

        {
          path: 'finance/cash',
          name: 'finance-cash',
          component: FinanceCashView,

          meta: {
            roles: [
              'admin',
              'finance',
            ],
          },
        },

        {
          path: 'finance/profit',
          name: 'finance-profit',
          component: FinanceProfitView,

          meta: {
            roles: [
              'admin',
              'finance',
            ],
          },
        },

        {
          path: 'finance/tax',
          name: 'finance-tax',
          component: FinanceTaxView,

          meta: {
            roles: [
              'admin',
              'finance',
            ],
          },
        },

        {
          path: 'finance/reports',
          name: 'finance-reports',
          component: FinanceReportsView,

          meta: {
            roles: [
              'admin',
              'finance',
            ],
          },
        },

        {
          path: 'reports',
          name: 'reports',
          component: ReportsView,

          meta: {
            roles: [
              'admin',
              'finance',
            ],
          },
        },

        {
          path: 'users',
          name: 'users',
          component: UsersView,

          meta: {
            roles: [
              'admin',
            ],
          },
        },

        {
          path: 'profile',
          name: 'profile',
          component: ProfileView,

          meta: {
            roles: [
              'admin',
              'pegawai',
              'finance',
            ],
          },
        },

        {
          path: 'settings/company',
          name: 'company-settings',
          component: CompanySettingsView,

          meta: {
            roles: [
              'admin',
              'finance',
              'pegawai',
            ],
          },
        },

        {
          path: 'settings',
          name: 'settings',
          component: SettingsView,

          meta: {
            roles: [
              'admin',
              'pegawai',
              'finance',
            ],
          },
        },
      ],
    },
  ],
})

router.beforeEach((to) => {
  const token = localStorage.getItem('fas_token')

  const rawUser = localStorage.getItem('fas_user')

  let user = null

  if (rawUser) {
    try {
      user = JSON.parse(rawUser)
    } catch {
      localStorage.removeItem('fas_user')
    }
  }

  const recognizedRole = [
    'admin',
    'pegawai',
    'finance',
  ].includes(user?.role)

  const needsAuth = to.matched.some(
    (record) => record.meta.requiresAuth,
  )

  if (
    needsAuth &&
    (!token || !recognizedRole)
  ) {
    return {
      path: '/login',
      query: {
        redirect: to.fullPath,
      },
    }
  }

  if (
    to.path === '/login' &&
    token &&
    recognizedRole
  ) {
    return homeForRole(user.role)
  }

  const allowedRoles = to.meta.roles

  if (
    Array.isArray(allowedRoles) &&
    !allowedRoles.includes(user?.role)
  ) {
    return homeForRole(user?.role)
  }

  return true
})

export default router
