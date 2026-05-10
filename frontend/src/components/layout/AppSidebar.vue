<template>
  <aside class="sidebar" :class="{ 'sidebar--collapsed': collapsed }">
    <!-- Logo -->
    <div class="sidebar__brand">
      <div class="sidebar__logo">
        <KinoMark />
      </div>
      <Transition name="fade">
        <span v-if="!collapsed" class="sidebar__wordmark">kino</span>
      </Transition>
    </div>

    <!-- Main nav -->
    <nav class="sidebar__nav">
      <NavItem
        v-for="item in mainNav"
        :key="item.name"
        :item="item"
        :collapsed="collapsed"
      />
    </nav>

    <div class="sidebar__spacer" />

    <!-- Bottom nav -->
    <nav class="sidebar__nav sidebar__nav--bottom">
      <NavItem
        v-for="item in bottomNav"
        :key="item.name"
        :item="item"
        :collapsed="collapsed"
      />
      <button class="sidebar__collapse-btn" @click="$emit('toggle')">
        <ChevronLeft v-if="!collapsed" :size="16" />
        <ChevronRight v-else :size="16" />
      </button>
    </nav>
  </aside>
</template>

<script setup lang="ts">
import { markRaw } from 'vue'
import {
  LayoutDashboard,
  ArrowLeftRight,
  PiggyBank,
  TrendingUp,
  Workflow,
  Landmark,
  Settings,
  ChevronLeft,
  ChevronRight,
} from 'lucide-vue-next'
import NavItem from './NavItem.vue'
import KinoMark from '../ui/KinoMark.vue'

defineEmits<{ toggle: [] }>()
defineProps<{ collapsed: boolean }>()

const mainNav = [
  { name: 'overview',      label: 'Overview',      icon: markRaw(LayoutDashboard), to: '/' },
  { name: 'transactions',  label: 'Transactions',  icon: markRaw(ArrowLeftRight),  to: '/transactions' },
  { name: 'budgets',       label: 'Budgets',       icon: markRaw(PiggyBank),       to: '/budgets' },
  { name: 'net-worth',     label: 'Net Worth',     icon: markRaw(TrendingUp),      to: '/net-worth' },
  { name: 'cash-flow',     label: 'Cash Flow',     icon: markRaw(Workflow),        to: '/cash-flow' },
]

const bottomNav = [
  { name: 'accounts', label: 'Accounts', icon: markRaw(Landmark),  to: '/accounts' },
  { name: 'settings', label: 'Settings', icon: markRaw(Settings),  to: '/settings' },
]
</script>

<style scoped>
.sidebar {
  width: var(--sidebar-width);
  min-width: var(--sidebar-width);
  height: 100vh;
  background: var(--color-surface-1);
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  padding: var(--space-4) var(--space-3);
  transition:
    width var(--duration-slow) var(--ease-in-out),
    min-width var(--duration-slow) var(--ease-in-out),
    background-color var(--duration-base) var(--ease-in-out),
    border-color var(--duration-base) var(--ease-in-out);
  overflow: hidden;
}

.sidebar--collapsed {
  width: var(--sidebar-width-collapsed);
  min-width: var(--sidebar-width-collapsed);
}

/* Brand */
.sidebar__brand {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-2) var(--space-6);
}

.sidebar__logo {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.sidebar__wordmark {
  font: var(--text-display-sm);
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text-primary);
  letter-spacing: 1px;
  white-space: nowrap;
}

/* Nav */
.sidebar__nav {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.sidebar__spacer { flex: 1; }

.sidebar__nav--bottom {
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border);
}

.sidebar__collapse-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 36px;
  border-radius: var(--radius-md);
  border: none;
  background: transparent;
  color: var(--color-text-tertiary);
  cursor: pointer;
  margin-top: var(--space-1);
  transition: background-color var(--duration-fast) var(--ease-out), color var(--duration-fast) var(--ease-out);
}

.sidebar__collapse-btn:hover {
  background: var(--color-surface-2);
  color: var(--color-text-secondary);
}

.fade-enter-active,
.fade-leave-active { transition: opacity var(--duration-fast) var(--ease-out); }
.fade-enter-from,
.fade-leave-to { opacity: 0; }
</style>
