<template>
  <div class="tx-row">
    <div class="tx-row__icon" :style="{ background: `${catColor}18`, color: catColor }">
      <component :is="catIcon" :size="15" />
    </div>
    <div class="tx-row__info">
      <p class="text-body tx-row__merchant">{{ transaction.payeeNormalized || transaction.payee || 'Unknown' }}</p>
      <p class="text-body-sm tx-row__meta">{{ formatDate(transaction.date) }}</p>
    </div>
    <div class="tx-row__right">
      <span
        class="tx-row__amount tabular-nums"
        :class="transaction.amountCents > 0 ? 'amount--positive' : ''"
      >
        {{ transaction.amountCents > 0 ? '+' : '-' }}${{ formatAbs(transaction.amountCents) }}
      </span>
      <span v-if="cat" class="badge tx-row__category" :style="{ background: `${cat.color}18`, color: cat.color }">
        {{ cat.name }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, markRaw } from 'vue'
import { ShoppingCart, Tv, Banknote, Car, Apple, Tag, Home, Heart, Ticket, ShoppingBag, PiggyBank, TrendingUp, ArrowLeftRight } from 'lucide-vue-next'
import type { models } from '../../../wailsjs/go/models'

type Transaction = models.Transaction
type Category    = models.Category

const props = defineProps<{
  transaction: Transaction
  categories:  Category[]
}>()

const iconMap: Record<string, any> = {
  'home':          markRaw(Home),
  'utensils':      markRaw(Apple),
  'tv':            markRaw(Tv),
  'banknote':      markRaw(Banknote),
  'car':           markRaw(Car),
  'heart-pulse':   markRaw(Heart),
  'ticket':        markRaw(Ticket),
  'shopping-bag':  markRaw(ShoppingBag),
  'piggy-bank':    markRaw(PiggyBank),
  'trending-up':   markRaw(TrendingUp),
  'arrow-left-right': markRaw(ArrowLeftRight),
  'tag':           markRaw(Tag),
}

const cat = computed(() =>
  props.transaction.categoryId
    ? props.categories.find(c => c.id === props.transaction.categoryId) ?? null
    : null
)

const catColor = computed(() => cat.value?.color ?? '#5A6B60')
const catIcon  = computed(() => iconMap[cat.value?.icon ?? 'tag'] ?? markRaw(Tag))

function formatAbs(cents: number) {
  return (Math.abs(cents) / 100).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function formatDate(iso: string) {
  if (!iso) return ''
  const d = new Date(iso)
  const today = new Date()
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)
  if (d.toDateString() === today.toDateString()) return 'Today'
  if (d.toDateString() === yesterday.toDateString()) return 'Yesterday'
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}
</script>

<style scoped>
.tx-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-2);
  border-radius: var(--radius-md);
  transition: background-color var(--duration-fast) var(--ease-out);
  cursor: default;
}
.tx-row:hover { background: var(--color-surface-2); }

.tx-row__icon {
  width: 32px; height: 32px;
  border-radius: var(--radius-md);
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}

.tx-row__info { flex: 1; min-width: 0; }

.tx-row__merchant {
  font-weight: 500;
  color: var(--color-text-primary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}

.tx-row__meta { color: var(--color-text-tertiary); margin-top: 1px; }

.tx-row__right {
  display: flex; flex-direction: column; align-items: flex-end;
  gap: var(--space-1); flex-shrink: 0;
}

.tx-row__amount {
  font: var(--text-body); font-weight: 600;
  color: var(--color-text-primary);
  font-variant-numeric: tabular-nums;
}

.tx-row__category { font-size: 11px; }
</style>
