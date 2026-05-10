import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  { path: '/setup',       name: 'setup',        component: () => import('../views/SetupView.vue') },
  { path: '/',            name: 'overview',     component: () => import('../views/OverviewView.vue') },
  { path: '/transactions',name: 'transactions', component: () => import('../views/TransactionsView.vue') },
  { path: '/budgets',     name: 'budgets',      component: () => import('../views/BudgetsView.vue') },
  { path: '/net-worth',   name: 'net-worth',    component: () => import('../views/NetWorthView.vue') },
  { path: '/cash-flow',   name: 'cash-flow',    component: () => import('../views/CashFlowView.vue') },
  { path: '/accounts',    name: 'accounts',     component: () => import('../views/AccountsView.vue') },
  { path: '/settings',    name: 'settings',     component: () => import('../views/SettingsView.vue') },
]

export const router = createRouter({
  history: createWebHashHistory(),
  routes,
})
