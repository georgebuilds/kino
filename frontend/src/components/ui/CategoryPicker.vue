<template>
  <!-- Floating popover — teleported to body so z-index is reliable -->
  <Teleport to="body">
    <div class="cp-backdrop" @click="$emit('close')" />
    <div
      class="cp-popover"
      :style="{ top: `${pos.y}px`, left: `${pos.x}px` }"
      role="listbox"
      aria-label="Pick a category"
    >
      <!-- Search -->
      <div class="cp-search">
        <Search :size="13" class="cp-search__icon" />
        <input
          ref="inputEl"
          v-model="query"
          class="cp-search__input"
          placeholder="Search…"
          autocomplete="off"
          @keydown.escape="$emit('close')"
          @keydown.enter.prevent="selectFirst"
        />
      </div>

      <!-- List -->
      <ul class="cp-list">
        <!-- Uncategorized option -->
        <li
          class="cp-item cp-item--none"
          :class="{ 'cp-item--active': modelValue == null }"
          @click="pick(null)"
        >
          <span class="cp-item__dot cp-item__dot--none" />
          <span class="cp-item__name">Uncategorized</span>
          <Check v-if="modelValue == null" :size="12" class="cp-item__check" />
        </li>

        <li
          v-for="cat in filtered"
          :key="cat.id"
          class="cp-item"
          :class="{ 'cp-item--active': modelValue === cat.id }"
          @click="pick(cat.id)"
        >
          <span class="cp-item__dot" :style="{ background: cat.color }" />
          <span class="cp-item__name">{{ cat.name }}</span>
          <Check v-if="modelValue === cat.id" :size="12" class="cp-item__check" />
        </li>

        <li v-if="filtered.length === 0 && query" class="cp-empty">No match</li>
      </ul>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Search, Check } from 'lucide-vue-next'
import type { Category } from '../../stores/categories'

const props = defineProps<{
  categories: Category[]
  modelValue: number | null | undefined
  /** Anchor point (viewport coords) — popover positions itself relative to this */
  anchorX: number
  anchorY: number
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', id: number | null): void
  (e: 'close'): void
}>()

// ── Positioning ───────────────────────────────────────────────────────────────
const POPOVER_W = 220
const POPOVER_H = 280

const pos = computed(() => {
  const x = Math.min(props.anchorX, window.innerWidth  - POPOVER_W - 8)
  const y = Math.min(props.anchorY, window.innerHeight - POPOVER_H - 8)
  return { x: Math.max(8, x), y: Math.max(8, y) }
})

// ── Filter ────────────────────────────────────────────────────────────────────
const query = ref('')
const filtered = computed(() => {
  const q = query.value.toLowerCase().trim()
  return q
    ? props.categories.filter(c => c.name.toLowerCase().includes(q))
    : props.categories
})

// ── Actions ───────────────────────────────────────────────────────────────────
function pick(id: number | null) {
  emit('update:modelValue', id)
  emit('close')
}

function selectFirst() {
  if (filtered.value.length) pick(filtered.value[0].id)
  else if (query.value === '') pick(null)
}

// Auto-focus search input
const inputEl = ref<HTMLInputElement | null>(null)
onMounted(() => inputEl.value?.focus())
</script>

<style scoped>
.cp-backdrop {
  position: fixed; inset: 0; z-index: 299;
}

.cp-popover {
  position: fixed; z-index: 300;
  width: 220px;
  background: var(--color-surface-1);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: 0 8px 32px rgba(0,0,0,0.25);
  overflow: hidden;
  animation: pop-in 120ms var(--ease-out);
}

@keyframes pop-in {
  from { opacity: 0; transform: scale(0.95) translateY(-4px) }
  to   { opacity: 1; transform: scale(1)    translateY(0)     }
}

/* Search */
.cp-search {
  display: flex; align-items: center; gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--color-border);
}

.cp-search__icon { color: var(--color-text-tertiary); flex-shrink: 0; }

.cp-search__input {
  flex: 1; background: transparent; border: none; outline: none;
  font: var(--text-body-sm); color: var(--color-text-primary);
  min-width: 0;
}
.cp-search__input::placeholder { color: var(--color-text-tertiary); }

/* List */
.cp-list {
  list-style: none; margin: 0; padding: var(--space-1);
  max-height: 240px; overflow-y: auto;
}

.cp-item {
  display: flex; align-items: center; gap: var(--space-2);
  padding: var(--space-2) var(--space-2);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-out);
}

.cp-item:hover    { background: var(--color-surface-2); }
.cp-item--active  { background: rgba(26,138,97,0.08); }

.cp-item__dot {
  width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0;
}
.cp-item__dot--none {
  background: var(--color-surface-2);
  border: 1.5px solid var(--color-border);
}

.cp-item__name {
  flex: 1; font: var(--text-body-sm); color: var(--color-text-primary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}

.cp-item--none .cp-item__name { color: var(--color-text-tertiary); }

.cp-item__check { color: var(--color-primary); flex-shrink: 0; }

.cp-empty {
  padding: var(--space-3) var(--space-2);
  font: var(--text-body-sm); color: var(--color-text-tertiary);
  text-align: center;
}
</style>
