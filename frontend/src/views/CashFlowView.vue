<template>
  <div class="page">

    <!-- Header -->
    <div class="page-header flex items-center justify-between">
      <div>
        <h1 class="page-title">Cash Flow</h1>
        <p class="page-subtitle">Where your money comes from and goes</p>
      </div>
      <!-- Month nav -->
      <div class="month-nav">
        <button class="btn btn--ghost btn--icon-sm" @click="prevMonth" aria-label="Previous month">
          <ChevronLeft :size="16" />
        </button>
        <span class="month-nav__label">{{ monthLabel }}</span>
        <button
          class="btn btn--ghost btn--icon-sm"
          :disabled="isFutureMonth"
          @click="nextMonth"
          aria-label="Next month"
        >
          <ChevronRight :size="16" />
        </button>
      </div>
    </div>

    <!-- Stat strip -->
    <div class="cf-stats">
      <div class="cf-stat card">
        <p class="cf-stat__label">INCOME</p>
        <p class="cf-stat__value amount--positive tabular-nums">
          +{{ formatMoney(cashFlow?.incomeCents ?? 0) }}
        </p>
      </div>
      <div class="cf-stat card">
        <p class="cf-stat__label">SPENT</p>
        <p class="cf-stat__value tabular-nums">
          {{ formatMoney(cashFlow?.expenseCents ?? 0) }}
        </p>
      </div>
      <div class="cf-stat card">
        <p class="cf-stat__label">{{ savedLabel }}</p>
        <p
          class="cf-stat__value tabular-nums"
          :class="(cashFlow?.savedCents ?? 0) >= 0 ? 'amount--positive' : 'amount--negative'"
        >
          {{ (cashFlow?.savedCents ?? 0) >= 0 ? '+' : '' }}{{ formatMoney(cashFlow?.savedCents ?? 0) }}
        </p>
      </div>
      <div class="cf-stat card">
        <p class="cf-stat__label">SAVINGS RATE</p>
        <p class="cf-stat__value tabular-nums" :class="savingsRate >= 20 ? 'amount--positive' : ''">
          {{ savingsRate }}%
        </p>
      </div>
    </div>

    <!-- Loading skeleton -->
    <div v-if="loading" class="card cf-loading">
      <div class="skeleton-block" style="height:400px" />
    </div>

    <!-- Empty state -->
    <div v-else-if="isEmpty" class="card cf-empty">
      <div class="cf-empty__icon">
        <GitMerge :size="32" />
      </div>
      <h2 class="cf-empty__title">No data for {{ monthLabel }}</h2>
      <p class="cf-empty__sub">
        Add transactions or import a bank export to see your cash flow visualised here.
      </p>
    </div>

    <!-- Sankey card -->
    <div v-else class="card cf-chart-card">
      <!-- Column labels -->
      <div class="cf-col-labels">
        <span class="cf-col-label">Income</span>
        <span class="cf-col-label">Spending &amp; Savings</span>
      </div>

      <SankeyDiagram
        :left-nodes="cashFlow!.leftNodes ?? []"
        :right-nodes="cashFlow!.rightNodes ?? []"
        :links="cashFlow!.links ?? []"
      />

      <!-- Legend -->
      <div class="cf-legend">
        <div
          v-for="n in allNodes"
          :key="n.id"
          class="cf-legend__item"
        >
          <span class="cf-legend__dot" :style="{ background: n.color }" />
          <span class="cf-legend__name">{{ n.name }}</span>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ChevronLeft, ChevronRight, GitMerge } from 'lucide-vue-next'
import SankeyDiagram from '../components/ui/SankeyDiagram.vue'
import { GetCashFlow } from '../../wailsjs/go/main/App'
import type { main } from '../../wailsjs/go/models'

type CashFlow = main.CashFlow

// ── State ──────────────────────────────────────────────────────────────────────
const cashFlow = ref<CashFlow | null>(null)
const loading  = ref(false)
const error    = ref<string | null>(null)

const now   = new Date()
const year  = ref(now.getFullYear())
const month = ref(now.getMonth() + 1)

// ── Month navigation ───────────────────────────────────────────────────────────
const monthLabel = computed(() =>
  new Date(year.value, month.value - 1, 1)
    .toLocaleDateString('en-US', { month: 'long', year: 'numeric' })
)

const isFutureMonth = computed(() => {
  const n = new Date()
  return year.value > n.getFullYear() ||
    (year.value === n.getFullYear() && month.value >= n.getMonth() + 1)
})

function prevMonth() {
  if (month.value === 1) { month.value = 12; year.value-- }
  else month.value--
  load()
}

function nextMonth() {
  if (month.value === 12) { month.value = 1; year.value++ }
  else month.value++
  load()
}

// ── Data ───────────────────────────────────────────────────────────────────────
async function load() {
  loading.value  = true
  error.value    = null
  cashFlow.value = null
  try {
    cashFlow.value = await GetCashFlow(year.value, month.value)
  } catch (e: any) {
    error.value = e?.message ?? 'Failed to load cash flow'
  } finally {
    loading.value = false
  }
}

onMounted(load)

// ── Derived ────────────────────────────────────────────────────────────────────
const isEmpty = computed(() =>
  !cashFlow.value ||
  ((cashFlow.value.leftNodes?.length ?? 0) === 0 &&
   (cashFlow.value.rightNodes?.length ?? 0) === 0)
)

const savingsRate = computed(() => {
  const inc = cashFlow.value?.incomeCents ?? 0
  const sav = cashFlow.value?.savedCents  ?? 0
  if (inc <= 0) return 0
  return Math.max(0, Math.round((sav / inc) * 100))
})

const savedLabel = computed(() =>
  (cashFlow.value?.savedCents ?? 0) >= 0 ? 'SAVED' : 'DEFICIT'
)

const allNodes = computed(() => [
  ...(cashFlow.value?.leftNodes  ?? []),
  ...(cashFlow.value?.rightNodes ?? []),
])

// ── Formatting ─────────────────────────────────────────────────────────────────
function formatMoney(cents: number) {
  const abs = Math.abs(cents)
  return (cents < 0 ? '-$' : '$') +
    (abs / 100).toLocaleString('en-US', { minimumFractionDigits: 0, maximumFractionDigits: 0 })
}
</script>

<style scoped>
/* ── Month nav ── */
.month-nav {
  display: flex; align-items: center; gap: var(--space-1);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: 2px;
}

.month-nav__label {
  font: var(--text-body-sm); font-weight: 500;
  color: var(--color-text-primary);
  min-width: 120px; text-align: center;
}

.btn--icon-sm {
  width: 28px; height: 28px; padding: 0;
  display: flex; align-items: center; justify-content: center;
}

/* ── Stat strip ── */
.cf-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-4);
  margin-bottom: var(--space-6);
}

@media (max-width: 800px) {
  .cf-stats { grid-template-columns: repeat(2, 1fr); }
}

.cf-stat { padding: var(--space-4) var(--space-5); }

.cf-stat__label {
  font-size: 10px; font-weight: 700; letter-spacing: 0.08em;
  color: var(--color-text-tertiary);
  margin-bottom: var(--space-2);
}

.cf-stat__value {
  font-size: 22px; font-weight: 700;
  font-family: var(--font-display);
  color: var(--color-text-primary);
  letter-spacing: -0.3px;
}

.amount--positive { color: var(--color-income); }
.amount--negative { color: var(--color-expense); }

/* ── Chart card ── */
.cf-chart-card {
  padding: var(--space-6);
  padding-bottom: var(--space-5);
}

.cf-col-labels {
  display: flex; justify-content: space-between;
  padding: 0 148px;  /* match labelW in SankeyDiagram */
  margin-bottom: var(--space-4);
}

.cf-col-label {
  font-size: 11px; font-weight: 700; letter-spacing: 0.07em;
  text-transform: uppercase; color: var(--color-text-tertiary);
}

/* ── Legend ── */
.cf-legend {
  display: flex; flex-wrap: wrap; gap: var(--space-2) var(--space-4);
  margin-top: var(--space-5);
  padding-top: var(--space-4);
  border-top: 1px solid var(--color-border);
}

.cf-legend__item {
  display: flex; align-items: center; gap: var(--space-2);
}

.cf-legend__dot {
  width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0;
}

.cf-legend__name {
  font: var(--text-body-sm); color: var(--color-text-secondary);
}

/* ── Loading ── */
.cf-loading { padding: var(--space-4); }

.skeleton-block {
  border-radius: var(--radius-md);
  background: linear-gradient(90deg, var(--color-surface-2) 25%, var(--color-surface-1) 50%, var(--color-surface-2) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite;
}
@keyframes shimmer { to { background-position: -200% 0 } }

/* ── Empty ── */
.cf-empty {
  display: flex; flex-direction: column; align-items: center;
  text-align: center; gap: var(--space-4);
  padding: var(--space-12) var(--space-8);
}

.cf-empty__icon {
  width: 64px; height: 64px; border-radius: var(--radius-lg);
  background: var(--color-surface-2);
  display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary);
}

.cf-empty__title { font: var(--text-heading); color: var(--color-text-primary); }
.cf-empty__sub {
  font: var(--text-body); color: var(--color-text-secondary);
  max-width: 360px; line-height: 1.6;
}
</style>
