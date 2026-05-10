<template>
  <RouterLink
    :to="item.to"
    class="nav-item"
    :class="{ 'nav-item--collapsed': collapsed }"
    :title="collapsed ? item.label : undefined"
  >
    <component :is="item.icon" :size="18" class="nav-item__icon" />
    <Transition name="label">
      <span v-if="!collapsed" class="nav-item__label">{{ item.label }}</span>
    </Transition>
  </RouterLink>
</template>

<script setup lang="ts">
import type { Component } from 'vue'

defineProps<{
  item: { name: string; label: string; icon: Component; to: string }
  collapsed: boolean
}>()
</script>

<style scoped>
.nav-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  color: var(--color-text-secondary);
  font: var(--text-body);
  font-weight: 500;
  white-space: nowrap;
  border-left: 2px solid transparent;
  transition:
    background-color var(--duration-fast) var(--ease-out),
    color var(--duration-fast) var(--ease-out),
    border-color var(--duration-fast) var(--ease-out);
  text-decoration: none;
}

.nav-item--collapsed {
  justify-content: center;
  padding: var(--space-2);
  border-left-color: transparent !important;
}

.nav-item:hover {
  background: var(--color-surface-2);
  color: var(--color-text-primary);
}

.nav-item.router-link-active {
  background: var(--color-primary-50);
  color: var(--color-primary-600);
  border-left-color: var(--color-primary);
}

[data-theme="dark"] .nav-item.router-link-active {
  background: rgba(26, 138, 97, 0.12);
  color: var(--color-primary-300);
}

.nav-item__icon { flex-shrink: 0; }

.label-enter-active,
.label-leave-active { transition: opacity var(--duration-fast) var(--ease-out); }
.label-enter-from,
.label-leave-to { opacity: 0; }
</style>
