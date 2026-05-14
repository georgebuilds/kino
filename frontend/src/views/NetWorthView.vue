<template>
  <div class="page">

    <!-- Header -->
    <div class="page-header flex items-center justify-between">
      <div>
        <h1 class="page-title">Net Worth</h1>
        <p class="page-subtitle">Total assets minus liabilities over time</p>
      </div>
      <!-- Period toggle -->
      <div class="period-toggle">
        <button
          v-for="p in PERIODS"
          :key="p.months"
          class="period-btn"
          :class="{ 'period-btn--active': period === p.months }"
          @click="setPeriod(p.months)"
        >{{ p.label }}</button>
      </div>
    </div>

    <!-- ── Summary strip ─────────────────────────────────────────────────── -->
    <div v-if="!loading" class="nw-summary">
      <div class="nw-hero">
        <span class="nw-hero__label">Current net worth</span>
        <span class="nw-hero__value" :class="currentNetWorth < 0 ? 'amount--negative' : ''">
          {{ formatMoney(currentNetWorth) }}
        </span>
      </div>
      <div class="nw-deltas">
        <div class="nw-delta">
          <span class="nw-delta__label">This month</span>
          <span class="nw-delta__value" :class="momClass">
            <component :is="momIcon" :size="13" />
            {{ momFormatted }}
          </span>
        </div>
        <div v-if="hasYoY" class="nw-delta">
          <span class="nw-delta__label">This year</span>
          <span class="nw-delta__value" :class="yoyClass">
            <component :is="yoyIcon" :size="13" />
            {{ yoyFormatted }}
          </span>
        </div>
      </div>
    </div>

    <!-- ── Chart card ────────────────────────────────────────────────────── -->
    <div class="card nw-chart-card">
      <div v-if="loading" class="nw-chart-placeholder">
        <div class="skeleton-block" style="height: 200px" />
      </div>

      <div v-else-if="points.length === 0" class="nw-empty">
        <TrendingUp :size="28" style="color: var(--color-text-tertiary)" />
        <p>Import transactions to see your net worth history.</p>
      </div>

      <div v-else ref="chartWrap" class="nw-chart-wrap">
        <svg
          v-if="svgW > 0"
          :width="svgW"
          :height="SVG_H"
          class="nw-svg"
          role="img"
          aria-label="Net worth history chart"
          @mousemove="onMouseMove"
          @mouseleave="tooltip = null"
        >
          <title>Net worth history chart</title>
          <defs>
            <linearGradient id="nw-area-grad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%"   :stop-color="areaColor" stop-opacity="0.35" />
              <stop offset="100%" :stop-color="areaColor" stop-opacity="0.02" />
            </linearGradient>
          </defs>

          <!-- Y-axis grid lines -->
          <g class="nw-grid">
            <line
              v-for="tick in yTicks"
              :key="tick.val"
              :x1="MARGIN.l"
              :y1="tick.y"
              :x2="svgW - MARGIN.r"
              :y2="tick.y"
              class="nw-grid__line"
            />
            <text
              v-for="tick in yTicks"
              :key="'t' + tick.val"
              :x="MARGIN.l - 6"
              :y="tick.y + 4"
              class="nw-axis-label nw-axis-label--y"
            >{{ tick.label }}</text>
          </g>

          <!-- Zero line (if range spans negative) -->
          <line
            v-if="yMin < 0 && yMax > 0"
            :x1="MARGIN.l"
            :y1="yToSvg(0)"
            :x2="svgW - MARGIN.r"
            :y2="yToSvg(0)"
            class="nw-zero-line"
          />

          <!-- Area fill -->
          <path :d="areaPath" fill="url(#nw-area-grad)" />

          <!-- Line -->
          <path :d="linePath" class="nw-line" />

          <!-- X-axis labels -->
          <g class="nw-x-axis">
            <text
              v-for="(lbl, i) in xLabels"
              :key="i"
              :x="xPositions[i]"
              :y="SVG_H - MARGIN.b + 14"
              class="nw-axis-label nw-axis-label--x"
              :class="{ 'nw-axis-label--bold': lbl.isYearStart }"
            >{{ lbl.text }}</text>
          </g>

          <!-- Data dots (only shown near hover) -->
          <g v-if="tooltip">
            <circle
              :cx="tooltip.x"
              :cy="tooltip.y"
              r="5"
              class="nw-dot"
              :style="{ fill: areaColor }"
            />
          </g>
        </svg>

        <!-- Tooltip -->
        <div
          v-if="tooltip"
          class="nw-tooltip"
          :style="tooltipStyle"
        >
          <p class="nw-tooltip__month">{{ tooltip.monthLabel }}</p>
          <p class="nw-tooltip__value">{{ formatMoney(tooltip.netWorth) }}</p>
          <div class="nw-tooltip__row">
            <span class="nw-tooltip__key nw-tooltip__key--asset">Assets</span>
            <span>{{ formatMoney(tooltip.assets) }}</span>
          </div>
          <div v-if="tooltip.liabilities !== 0" class="nw-tooltip__row">
            <span class="nw-tooltip__key nw-tooltip__key--liab">Liabilities</span>
            <span>{{ formatMoney(tooltip.liabilities) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- ── Account breakdown ─────────────────────────────────────────────── -->
    <div v-if="!loading" class="nw-breakdown">

      <!-- Assets -->
      <div class="card nw-group">
        <div class="nw-group__header">
          <span class="nw-group__title">Assets</span>
          <span class="nw-group__total">{{ formatMoney(totalAssets) }}</span>
        </div>
        <div v-if="assetAccounts.length === 0" class="nw-group__empty">No asset accounts</div>
        <div v-for="acc in assetAccounts" :key="acc.id" class="nw-acc-row">
          <span class="nw-acc-row__type-dot" :class="'dot--' + acc.type" />
          <span class="nw-acc-row__name">{{ acc.name }}</span>
          <span class="nw-acc-row__inst" v-if="acc.institution">{{ acc.institution }}</span>
          <span class="nw-acc-row__balance">{{ formatMoney(acc.balanceCents) }}</span>
        </div>
      </div>

      <!-- Liabilities -->
      <div class="card nw-group">
        <div class="nw-group__header">
          <span class="nw-group__title">Liabilities</span>
          <span class="nw-group__total nw-group__total--liab">{{ formatMoney(totalLiabilities) }}</span>
        </div>
        <div v-if="liabilityAccounts.length === 0" class="nw-group__empty">No liability accounts</div>
        <div v-for="acc in liabilityAccounts" :key="acc.id" class="nw-acc-row">
          <span class="nw-acc-row__type-dot" :class="'dot--' + acc.type" />
          <span class="nw-acc-row__name">{{ acc.name }}</span>
          <span class="nw-acc-row__inst" v-if="acc.institution">{{ acc.institution }}</span>
          <span class="nw-acc-row__balance nw-acc-row__balance--liab">{{ formatMoney(acc.balanceCents) }}</span>
        </div>
      </div>

    </div>

  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { TrendingUp, TrendingDown, Minus } from 'lucide-vue-next'
import { useResizeObserver } from '@vueuse/core'
import { GetNetWorthHistory } from '../../wailsjs/go/main/App'
import { useAccountsStore } from '../stores/accounts'
import type { db } from '../../wailsjs/go/models'

type NetWorthPoint = db.NetWorthPoint

// ── Constants ─────────────────────────────────────────────────────────────────
const PERIODS = [
  { months: 12, label: '12m' },
  { months: 24, label: '24m' },
]

const SVG_H  = 240
const MARGIN = { t: 16, r: 16, b: 28, l: 72 }
const ASSET_TYPES     = new Set(['checking','savings','investment','cash','crypto','other'])
const LIABILITY_TYPES = new Set(['credit_card','loan'])

// ── Data ──────────────────────────────────────────────────────────────────────
const period  = ref(12)
const points  = ref<NetWorthPoint[]>([])
const loading = ref(false)
const error   = ref<string | null>(null)

const accountsStore = useAccountsStore()

async function load() {
  loading.value = true
  error.value   = null
  try {
    points.value = await GetNetWorthHistory(period.value)
  } catch (e: any) {
    error.value = e?.message ?? 'Failed to load net worth history'
  } finally {
    loading.value = false
  }
}

function setPeriod(m: number) {
  period.value = m
  load()
}

onMounted(() => {
  if (!accountsStore.initialised) accountsStore.fetch()
  load()
})

// ── Derived account lists ─────────────────────────────────────────────────────
const assetAccounts = computed(() =>
  accountsStore.accounts.filter(a => ASSET_TYPES.has(a.type))
    .sort((a, b) => b.balanceCents - a.balanceCents)
)
const liabilityAccounts = computed(() =>
  accountsStore.accounts.filter(a => LIABILITY_TYPES.has(a.type))
    .sort((a, b) => a.balanceCents - b.balanceCents)
)
const totalAssets = computed(() =>
  assetAccounts.value.reduce((s, a) => s + a.balanceCents, 0)
)
const totalLiabilities = computed(() =>
  liabilityAccounts.value.reduce((s, a) => s + a.balanceCents, 0)
)

// ── Summary stats ─────────────────────────────────────────────────────────────
const currentNetWorth = computed(() =>
  points.value.length ? points.value[points.value.length - 1].netWorth : 0
)

const momDelta = computed(() => {
  if (points.value.length < 2) return 0
  const last = points.value[points.value.length - 1]
  const prev = points.value[points.value.length - 2]
  return last.netWorth - prev.netWorth
})

const yoyDelta = computed(() => {
  if (points.value.length < 13) return null
  const last = points.value[points.value.length - 1]
  const yearAgo = points.value[points.value.length - 13]
  return last.netWorth - yearAgo.netWorth
})

const hasYoY = computed(() => yoyDelta.value !== null)

function deltaClass(d: number) {
  if (d > 0)  return 'delta--up'
  if (d < 0)  return 'delta--down'
  return 'delta--flat'
}
function deltaIcon(d: number) {
  if (d > 0) return TrendingUp
  if (d < 0) return TrendingDown
  return Minus
}
function formatDelta(d: number) {
  const sign = d > 0 ? '+' : ''
  return sign + formatMoney(d)
}

const momClass    = computed(() => deltaClass(momDelta.value))
const momIcon     = computed(() => deltaIcon(momDelta.value))
const momFormatted = computed(() => formatDelta(momDelta.value))
const yoyClass    = computed(() => deltaClass(yoyDelta.value ?? 0))
const yoyIcon     = computed(() => deltaIcon(yoyDelta.value ?? 0))
const yoyFormatted = computed(() => formatDelta(yoyDelta.value ?? 0))

// ── Chart geometry ────────────────────────────────────────────────────────────
const chartWrap = ref<HTMLElement | null>(null)
const svgW      = ref(0)

useResizeObserver(chartWrap, entries => {
  svgW.value = entries[0].contentRect.width
})

// chart inner bounds
const innerW = computed(() => svgW.value - MARGIN.l - MARGIN.r)
const innerH = computed(() => SVG_H - MARGIN.t - MARGIN.b)

// Y range
const yMin = computed(() => {
  if (!points.value.length) return 0
  const values = points.value.map(p => p.netWorth)
  const mn = values.reduce((a, b) => Math.min(a, b), Infinity)
  return mn < 0 ? mn * 1.1 : Math.min(0, mn * 0.9)
})
const yMax = computed(() => {
  if (!points.value.length) return 1
  const values = points.value.map(p => p.netWorth)
  const mx = values.reduce((a, b) => Math.max(a, b), -Infinity)
  return mx * 1.1 || 1
})

function yToSvg(val: number): number {
  const range = yMax.value - yMin.value || 1
  return MARGIN.t + innerH.value * (1 - (val - yMin.value) / range)
}

// X positions
const xPositions = computed(() => {
  const n = points.value.length
  if (n < 2) return points.value.map(() => MARGIN.l + innerW.value / 2)
  return points.value.map((_, i) => MARGIN.l + (i / (n - 1)) * innerW.value)
})

// SVG paths
const linePath = computed(() => {
  if (!points.value.length || svgW.value === 0) return ''
  return points.value
    .map((p, i) => `${i === 0 ? 'M' : 'L'}${xPositions.value[i].toFixed(1)},${yToSvg(p.netWorth).toFixed(1)}`)
    .join(' ')
})

const areaPath = computed(() => {
  if (!points.value.length || svgW.value === 0) return ''
  const baseline = yToSvg(Math.max(0, yMin.value))
  const last = points.value.length - 1
  return linePath.value +
    ` L${xPositions.value[last].toFixed(1)},${baseline.toFixed(1)}` +
    ` L${xPositions.value[0].toFixed(1)},${baseline.toFixed(1)} Z`
})

// Colour: green if net positive, red if negative
const areaColor = computed(() =>
  currentNetWorth.value >= 0 ? '#1A8A61' : '#EF4444'
)

// Y-axis ticks
const yTicks = computed(() => {
  const range  = yMax.value - yMin.value || 1
  const rough  = range / 4
  const mag    = Math.pow(10, Math.floor(Math.log10(Math.abs(rough))))
  const step   = Math.ceil(rough / mag) * mag || mag
  const ticks: { val: number; y: number; label: string }[] = []
  let v = Math.floor(yMin.value / step) * step
  while (v <= yMax.value + step * 0.5) {
    if (v >= yMin.value - step * 0.1) {
      ticks.push({ val: v, y: yToSvg(v), label: formatCompact(v) })
    }
    v += step
  }
  return ticks
})

// X-axis labels (skip some when dense)
const xLabels = computed(() => {
  const n    = points.value.length
  const skip = n > 18 ? 3 : n > 12 ? 2 : 1
  return points.value.map((p, i) => {
    const [y, m] = p.month.split('-').map(Number)
    const show = i % skip === 0 || i === n - 1
    const text = show
      ? m === 1
        ? String(y)  // show year on January
        : new Date(y, m - 1).toLocaleString('en-US', { month: 'short' })
      : ''
    return { text, isYearStart: m === 1 }
  })
})

// ── Tooltip ───────────────────────────────────────────────────────────────────
interface Tooltip {
  x: number; y: number
  monthLabel: string
  netWorth: number; assets: number; liabilities: number
}
const tooltip = ref<Tooltip | null>(null)

function onMouseMove(e: MouseEvent) {
  if (!points.value.length || svgW.value === 0) return
  const rect  = (e.currentTarget as SVGElement).getBoundingClientRect()
  const mx    = e.clientX - rect.left
  // Find nearest point
  let best = 0, bestDist = Infinity
  for (let i = 0; i < xPositions.value.length; i++) {
    const d = Math.abs(xPositions.value[i] - mx)
    if (d < bestDist) { bestDist = d; best = i }
  }
  const p = points.value[best]
  const [y, m] = p.month.split('-').map(Number)
  const monthLabel = new Date(y, m - 1).toLocaleString('en-US', { month: 'long', year: 'numeric' })
  tooltip.value = {
    x:          xPositions.value[best],
    y:          yToSvg(p.netWorth),
    monthLabel,
    netWorth:    p.netWorth,
    assets:      p.assets,
    liabilities: p.liabilities,
  }
}

const tooltipStyle = computed(() => {
  if (!tooltip.value || svgW.value === 0) return {}
  const left = tooltip.value.x > svgW.value / 2
    ? `${tooltip.value.x - 150}px`
    : `${tooltip.value.x + 12}px`
  return { left, top: `${tooltip.value.y - 20}px` }
})

// ── Formatters ────────────────────────────────────────────────────────────────
function formatMoney(cents: number): string {
  const abs = Math.abs(cents)
  const pfx = cents < 0 ? '-$' : '$'
  return pfx + (abs / 100).toLocaleString('en-US', { minimumFractionDigits: 0, maximumFractionDigits: 0 })
}

function formatCompact(cents: number): string {
  const abs = Math.abs(cents)
  const pfx = cents < 0 ? '-$' : '$'
  if (abs >= 1_000_000_00) return pfx + (abs / 1_000_000_00).toFixed(1) + 'M'
  if (abs >= 1_000_00)     return pfx + (abs / 1_000_00).toFixed(0)     + 'k'
  return pfx + (abs / 100).toFixed(0)
}
</script>

<style scoped>
/* ── Summary ── */
.nw-summary {
  display: flex;
  align-items: flex-end;
  gap: var(--space-6);
  margin-bottom: var(--space-5);
  flex-wrap: wrap;
}
.nw-hero {
  display: flex; flex-direction: column; gap: 2px;
}
.nw-hero__label {
  font-size: 11px; font-weight: 600; letter-spacing: 0.07em; text-transform: uppercase;
  color: var(--color-text-tertiary);
}
.nw-hero__value {
  font-size: 36px; font-weight: 700; letter-spacing: -0.03em;
  color: var(--color-text-primary);
  line-height: 1;
}
.nw-deltas {
  display: flex; gap: var(--space-5); padding-bottom: 4px;
}
.nw-delta {
  display: flex; flex-direction: column; gap: 2px;
}
.nw-delta__label {
  font-size: 11px; font-weight: 600; letter-spacing: 0.07em; text-transform: uppercase;
  color: var(--color-text-tertiary);
}
.nw-delta__value {
  display: flex; align-items: center; gap: 4px;
  font-size: 16px; font-weight: 600;
}
.delta--up   { color: var(--color-income); }
.delta--down { color: var(--color-expense); }
.delta--flat { color: var(--color-text-secondary); }
.amount--negative { color: var(--color-expense); }

/* Period toggle */
.period-toggle {
  display: flex; background: var(--color-surface-2);
  border-radius: var(--radius-md); padding: 3px; gap: 2px;
}
.period-btn {
  padding: var(--space-1) var(--space-3);
  border-radius: calc(var(--radius-md) - 2px);
  border: none; background: transparent;
  color: var(--color-text-secondary);
  font: var(--text-body-sm); font-weight: 500; cursor: pointer;
  transition: background var(--duration-fast) var(--ease-out),
              color       var(--duration-fast) var(--ease-out),
              box-shadow  var(--duration-fast) var(--ease-out);
}
.period-btn--active {
  background: var(--color-surface-1); color: var(--color-text-primary);
  box-shadow: var(--shadow-sm);
}

/* ── Chart card ── */
.nw-chart-card { padding: var(--space-4); margin-bottom: var(--space-5); }

.nw-chart-wrap { position: relative; }

.nw-svg { display: block; overflow: visible; }

/* Grid */
.nw-grid__line {
  stroke: var(--color-border); stroke-width: 1; stroke-dasharray: 3 4;
}
.nw-zero-line {
  stroke: var(--color-border); stroke-width: 1.5;
}
.nw-axis-label {
  fill: var(--color-text-tertiary);
  font-size: 10px; font-family: inherit;
}
.nw-axis-label--y { text-anchor: end; }
.nw-axis-label--x { text-anchor: middle; }
.nw-axis-label--bold { font-weight: 700; }

/* Line */
.nw-line {
  fill: none;
  stroke: v-bind(areaColor);
  stroke-width: 2.5;
  stroke-linejoin: round;
  stroke-linecap: round;
}

.nw-dot {
  stroke: var(--color-surface-1); stroke-width: 2;
}

/* Tooltip */
.nw-tooltip {
  position: absolute;
  background: var(--color-surface-1);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  min-width: 140px;
  box-shadow: var(--shadow-md);
  pointer-events: none;
  z-index: 10;
}
.nw-tooltip__month {
  font-size: 11px; font-weight: 600; text-transform: uppercase;
  letter-spacing: 0.06em; color: var(--color-text-tertiary);
  margin-bottom: var(--space-1);
}
.nw-tooltip__value {
  font-size: 18px; font-weight: 700; color: var(--color-text-primary);
  margin-bottom: var(--space-2);
}
.nw-tooltip__row {
  display: flex; justify-content: space-between; gap: var(--space-3);
  font: var(--text-body-sm);
}
.nw-tooltip__key { color: var(--color-text-tertiary); }
.nw-tooltip__key--asset { color: var(--color-income); }
.nw-tooltip__key--liab  { color: var(--color-expense); }

/* Empty / loading */
.nw-chart-placeholder { padding: var(--space-4) 0; }
.skeleton-block {
  border-radius: var(--radius-md);
  background: linear-gradient(90deg,
    var(--color-surface-2) 25%, var(--color-surface-1) 50%, var(--color-surface-2) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite;
}
@keyframes shimmer { to { background-position: -200% 0 } }

.nw-empty {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: var(--space-3); height: 160px;
  color: var(--color-text-secondary); font: var(--text-body);
}

/* ── Breakdown ── */
.nw-breakdown {
  display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-4);
}
@media (max-width: 640px) {
  .nw-breakdown { grid-template-columns: 1fr; }
}

.nw-group { padding: 0; overflow: hidden; }
.nw-group__header {
  display: flex; justify-content: space-between; align-items: center;
  padding: var(--space-4) var(--space-4) var(--space-3);
  border-bottom: 1px solid var(--color-border);
}
.nw-group__title {
  font-size: 12px; font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase;
  color: var(--color-text-tertiary);
}
.nw-group__total {
  font: var(--text-body); font-weight: 700; color: var(--color-income);
}
.nw-group__total--liab { color: var(--color-expense); }
.nw-group__empty {
  padding: var(--space-4); color: var(--color-text-tertiary); font: var(--text-body-sm);
}

.nw-acc-row {
  display: flex; align-items: center; gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--color-border);
}
.nw-acc-row:last-child { border-bottom: none; }

.nw-acc-row__type-dot {
  width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0;
}
/* Account type colours */
.dot--checking   { background: #2A7FA8; }
.dot--savings    { background: #1A8A61; }
.dot--investment { background: #147050; }
.dot--cash       { background: #4A9E8A; }
.dot--crypto     { background: #8B5CF6; }
.dot--other      { background: #5A6B60; }
.dot--credit_card { background: #B84A72; }
.dot--loan       { background: #C4603A; }

.nw-acc-row__name {
  flex: 1; font: var(--text-body); min-width: 0;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.nw-acc-row__inst {
  font: var(--text-body-sm); color: var(--color-text-tertiary);
  flex-shrink: 0;
}
.nw-acc-row__balance {
  font: var(--text-body); font-weight: 600; tabular-nums: true;
  color: var(--color-text-primary); flex-shrink: 0;
}
.nw-acc-row__balance--liab { color: var(--color-expense); }
</style>
