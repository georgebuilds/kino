<template>
  <div class="page">
    <!-- Header -->
    <div class="page-header flex items-center justify-between">
      <div>
        <h1 class="page-title">Good {{ timeOfDay }}</h1>
        <p class="page-subtitle">{{ currentMonth }}</p>
      </div>
      <button class="btn btn--ghost btn--sm" @click="refresh" :disabled="summaryLoading">
        <RefreshCw :size="14" :class="{ 'spin': summaryLoading }" />
        Sync
      </button>
    </div>

    <!-- Net worth hero -->
    <div class="card card--lg overview__hero">
      <p class="text-label" style="color: var(--color-text-secondary); margin-bottom: var(--space-3);">
        NET WORTH
      </p>
      <div class="amount-display" style="margin-bottom: var(--space-3);">
        <span class="currency-symbol">{{ (summary?.netWorthCents ?? 0) < 0 ? '-$' : '$' }}</span>
        <span class="amount-value">{{ formatWhole(summary?.netWorthCents ?? 0) }}</span>
        <span class="amount-cents">.{{ formatCents(summary?.netWorthCents ?? 0) }}</span>
      </div>
      <div class="flex items-center gap-2">
        <span
          v-if="summary"
          class="badge"
          :class="(summary.netWorthDeltaCents ?? 0) >= 0 ? 'badge--income' : 'badge--expense'"
        >
          <TrendingUp v-if="(summary.netWorthDeltaCents ?? 0) >= 0" :size="11" style="margin-right:3px" />
          <TrendingDown v-else :size="11" style="margin-right:3px" />
          {{ (summary.netWorthDeltaCents ?? 0) >= 0 ? '+' : '' }}{{ formatMoney(summary.netWorthDeltaCents ?? 0) }} this month
        </span>
        <span v-if="summaryLoading" class="text-body-sm" style="color:var(--color-text-tertiary)">
          Loading…
        </span>
      </div>
    </div>

    <!-- Stat cards -->
    <div class="page-grid page-grid--3" style="margin-top: var(--space-6);">
      <StatCard
        label="Spent this month"
        :value="summary?.expenseCents ?? 0"
        direction="expense"
        icon="credit-card"
      />
      <StatCard
        label="Saved this month"
        :value="summary?.savedCents ?? 0"
        direction="income"
        icon="piggy-bank"
      />
      <StatCard
        label="Top category"
        :value="summary?.topCategoryCents ?? 0"
        :caption="summary?.topCategory || '—'"
        direction="neutral"
        icon="tag"
      />
    </div>

    <!-- Accounts + Recent transactions -->
    <div class="overview__bottom" style="margin-top: var(--space-6);">
      <!-- Account balances -->
      <div class="card">
        <div class="flex items-center justify-between" style="margin-bottom: var(--space-5);">
          <h2 class="text-heading">Accounts</h2>
          <RouterLink to="/accounts" class="btn btn--ghost btn--sm">Manage</RouterLink>
        </div>
        <div v-if="accountsStore.loading" class="overview__empty">Loading…</div>
        <div v-else-if="accountsStore.accounts.length === 0" class="overview__empty">
          No accounts yet — <RouterLink to="/accounts">add one</RouterLink>
        </div>
        <div v-else class="account-list">
          <AccountRow
            v-for="acc in accountsStore.accounts"
            :key="acc.id"
            :account="acc"
          />
        </div>
      </div>

      <!-- Recent transactions -->
      <div class="card">
        <div class="flex items-center justify-between" style="margin-bottom: var(--space-5);">
          <h2 class="text-heading">Recent</h2>
          <RouterLink to="/transactions" class="btn btn--ghost btn--sm">All transactions</RouterLink>
        </div>
        <div v-if="txStore.loading" class="overview__empty">Loading…</div>
        <div v-else-if="txStore.transactions.length === 0" class="overview__empty">
          No transactions yet
        </div>
        <div v-else class="transaction-list">
          <TransactionRow
            v-for="tx in recentTxs"
            :key="tx.id"
            :transaction="tx"
            :categories="categoriesStore.categories"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { RefreshCw, TrendingUp, TrendingDown } from 'lucide-vue-next'
import StatCard from '../components/ui/StatCard.vue'
import AccountRow from '../components/ui/AccountRow.vue'
import TransactionRow from '../components/ui/TransactionRow.vue'
import { useAccountsStore } from '../stores/accounts'
import { useCategoriesStore } from '../stores/categories'
import { useTransactionsStore } from '../stores/transactions'
import { GetMonthSummary } from '../../wailsjs/go/main/App'
import type { main } from '../../wailsjs/go/models'

// ── Stores ────────────────────────────────────────────────────────────────
const accountsStore   = useAccountsStore()
const categoriesStore = useCategoriesStore()
const txStore         = useTransactionsStore()

// ── Summary ───────────────────────────────────────────────────────────────
const summary        = ref<main.MonthSummary | null>(null)
const summaryLoading = ref(false)

const now = new Date()

async function loadSummary() {
  summaryLoading.value = true
  try {
    summary.value = await GetMonthSummary(now.getFullYear(), now.getMonth() + 1)
  } catch (e: any) {
    console.error('Failed to load summary:', e?.message ?? e)
  } finally {
    summaryLoading.value = false
  }
}

async function refresh() {
  await Promise.all([
    accountsStore.fetch(),
    txStore.fetch(),
    loadSummary(),
  ])
}

const originalLimit = txStore.filter.limit
onMounted(() => {
  txStore.filter.limit  = 10
  txStore.filter.offset = 0
  txStore.fetch()
  loadSummary()
})
onUnmounted(() => { txStore.filter.limit = originalLimit })

// ── Derived ───────────────────────────────────────────────────────────────
const recentTxs = computed(() => txStore.transactions.slice(0, 10))

// ── Formatting ────────────────────────────────────────────────────────────
const hour = now.getHours()
const timeOfDay = hour < 12 ? 'morning' : hour < 17 ? 'afternoon' : 'evening'
const currentMonth = now.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })

function centsToAbs(cents: number) { return Math.abs(cents) }

function formatWhole(cents: number) {
  return Math.floor(centsToAbs(cents) / 100).toLocaleString()
}

function formatCents(cents: number) {
  return String(centsToAbs(cents) % 100).padStart(2, '0')
}

function formatMoney(cents: number) {
  const abs = centsToAbs(cents)
  return '$' + (abs / 100).toLocaleString('en-US', { minimumFractionDigits: 0, maximumFractionDigits: 0 })
}
</script>

<style scoped>
.overview__hero {
  background: linear-gradient(135deg, var(--color-surface-1) 0%, var(--color-surface-2) 100%);
  margin-top: var(--space-2);
}

.overview__bottom {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-6);
  padding-bottom: var(--space-8);
}

.account-list,
.transaction-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.overview__empty {
  font: var(--text-body-sm);
  color: var(--color-text-tertiary);
  padding: var(--space-4) 0;
}

.spin { animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

@media (max-width: 800px) {
  .overview__bottom { grid-template-columns: 1fr; }
}
</style>
