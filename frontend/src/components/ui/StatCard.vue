<template>
  <div class="card stat-card">
    <div class="flex items-center justify-between" style="margin-bottom: var(--space-4);">
      <p class="text-label" style="color: var(--color-text-secondary);">{{ label.toUpperCase() }}</p>
      <component :is="iconComponent" :size="16" class="stat-card__icon" />
    </div>

    <div class="stat-card__value" :class="`stat-card__value--${direction}`">
      <span class="stat-card__currency">$</span>
      <span class="tabular-nums">{{ formatNumber(value) }}</span>
    </div>

    <div v-if="of" class="stat-card__bar-wrap">
      <div class="stat-card__bar">
        <div class="stat-card__bar-fill" :style="{ width: barWidth, background: barColor }" />
      </div>
      <p class="text-body-sm" style="color: var(--color-text-secondary); margin-top: var(--space-2);">
        of ${{ formatNumber(of) }} budget
      </p>
    </div>

    <p v-if="caption" class="text-body-sm" style="color: var(--color-text-secondary); margin-top: var(--space-2);">
      {{ caption }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, markRaw } from 'vue'
import { CreditCard, PiggyBank, Tag } from 'lucide-vue-next'

// value and of are in CENTS
const props = defineProps<{
  label: string
  value: number  // cents
  of?: number    // cents budget cap
  direction: 'income' | 'expense' | 'neutral'
  icon: 'credit-card' | 'piggy-bank' | 'tag'
  caption?: string
}>()

const iconMap = {
  'credit-card': markRaw(CreditCard),
  'piggy-bank':  markRaw(PiggyBank),
  'tag':         markRaw(Tag),
}
const iconComponent = computed(() => iconMap[props.icon])

function formatNumber(cents: number) {
  return (Math.abs(cents) / 100).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

const ratio = computed(() => props.of ? Math.min(props.value / props.of, 1) : 0)
const barWidth = computed(() => `${ratio.value * 100}%`)
const barColor = computed(() => {
  if (!props.of) return 'var(--color-primary)'
  if (ratio.value < 0.75) return 'var(--color-primary)'
  if (ratio.value < 0.90) return 'var(--color-gold-500)'
  if (ratio.value < 1.0)  return 'var(--color-warning)'
  return 'var(--color-expense)'
})
</script>

<style scoped>
.stat-card { position: relative; overflow: hidden; }

.stat-card__icon { color: var(--color-text-tertiary); }

.stat-card__value {
  display: flex;
  align-items: baseline;
  gap: 2px;
  font-size: 28px;
  font-weight: 700;
  font-family: var(--font-display);
  letter-spacing: -0.5px;
  line-height: 1;
  margin-bottom: var(--space-3);
  font-variant-numeric: tabular-nums;
}

.stat-card__currency {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.stat-card__value--income  { color: var(--color-income); }
.stat-card__value--expense { color: var(--color-text-primary); }
.stat-card__value--neutral { color: var(--color-text-primary); }

.stat-card__bar-wrap { margin-top: var(--space-1); }

.stat-card__bar {
  height: 4px;
  background: var(--color-surface-2);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.stat-card__bar-fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width 600ms var(--ease-out), background-color var(--duration-base) var(--ease-out);
}
</style>
