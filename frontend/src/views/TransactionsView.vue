<template>
  <div class="page">

    <!-- Header -->
    <div class="page-header flex items-center justify-between">
      <div>
        <h1 class="page-title">Transactions</h1>
        <p class="page-subtitle">
          {{ txStore.total.toLocaleString() }} transaction{{ txStore.total === 1 ? '' : 's' }}
          <span v-if="hasActiveFilters" class="filter-badge">filtered</span>
        </p>
      </div>
      <div class="flex items-center gap-2">
        <button class="btn btn--ghost btn--sm" @click="showImportModal = true">
          <Upload :size="14" />
          Import
        </button>
        <button class="btn btn--primary btn--sm" @click="openCreate">
          <Plus :size="14" />
          Add transaction
        </button>
      </div>
    </div>

    <!-- Filter bar -->
    <div class="filter-bar card">
      <!-- Search -->
      <div class="filter-bar__search">
        <Search :size="14" class="filter-bar__search-icon" />
        <input
          v-model="searchDraft"
          class="filter-bar__search-input"
          placeholder="Search payee…"
          autocomplete="off"
          @input="onSearchInput"
          @keydown.escape="clearSearch"
        />
        <button v-if="searchDraft" class="filter-bar__clear-btn" @click="clearSearch" aria-label="Clear search">
          <X :size="12" />
        </button>
      </div>

      <!-- Account -->
      <select v-model="filterAccountId" class="filter-bar__select" @change="applyFilters">
        <option :value="undefined">All accounts</option>
        <option v-for="acc in accountsStore.accounts" :key="acc.id" :value="acc.id">{{ acc.name }}</option>
      </select>

      <!-- Category -->
      <select v-model="filterCategoryId" class="filter-bar__select" @change="applyFilters">
        <option :value="undefined">All categories</option>
        <option v-for="cat in categoriesStore.categories" :key="cat.id" :value="cat.id">{{ cat.name }}</option>
      </select>

      <!-- Date from -->
      <input
        v-model="filterDateFrom"
        class="filter-bar__date"
        type="date"
        title="From date"
        @change="applyFilters"
      />
      <span class="filter-bar__date-sep">–</span>
      <!-- Date to -->
      <input
        v-model="filterDateTo"
        class="filter-bar__date"
        type="date"
        title="To date"
        @change="applyFilters"
      />

      <!-- Clear all -->
      <button
        v-if="hasActiveFilters"
        class="btn btn--ghost btn--sm"
        @click="clearAllFilters"
        title="Clear all filters"
      >
        <FilterX :size="14" />
      </button>
    </div>

    <!-- Loading skeleton -->
    <div v-if="txStore.loading && txStore.transactions.length === 0" class="tx-skeleton">
      <div v-for="i in 8" :key="i" class="skeleton-row" />
    </div>

    <!-- Empty state -->
    <div v-else-if="!txStore.loading && txStore.transactions.length === 0" class="tx-empty card">
      <div class="tx-empty__icon">
        <Receipt :size="28" />
      </div>
      <h2 class="tx-empty__title">{{ hasActiveFilters ? 'No matching transactions' : 'No transactions yet' }}</h2>
      <p class="tx-empty__sub">
        {{ hasActiveFilters
          ? 'Try adjusting your search or filters.'
          : 'Add transactions manually or import a CSV / OFX file.' }}
      </p>
      <div class="flex items-center gap-2">
        <button v-if="hasActiveFilters" class="btn btn--ghost" @click="clearAllFilters">Clear filters</button>
        <button v-else class="btn btn--primary" @click="openCreate">
          <Plus :size="14" />
          Add transaction
        </button>
      </div>
    </div>

    <!-- Date-grouped list -->
    <div v-else class="tx-groups">
      <section
        v-for="group in groups"
        :key="group.date"
        class="tx-group"
      >
        <!-- Date header -->
        <div class="tx-group__header">
          <span class="tx-group__date">{{ formatGroupDate(group.date) }}</span>
          <span class="tx-group__total tabular-nums" :class="group.net >= 0 ? 'amount--positive' : ''">
            {{ group.net >= 0 ? '+' : '' }}{{ formatMoney(group.net) }}
          </span>
        </div>

        <!-- Rows -->
        <div class="card tx-card">
          <div
            v-for="(tx, idx) in group.transactions"
            :key="tx.id"
            class="tx-card__row"
            :class="{ 'tx-card__row--last': idx === group.transactions.length - 1 }"
          >
            <!-- Existing TransactionRow for display -->
            <TransactionRow
              :transaction="tx"
              :categories="categoriesStore.categories"
            />

            <!-- Hover overlay: category + actions -->
            <div class="tx-card__overlay">
              <!-- Quick-assign category -->
              <button
                class="btn btn--ghost btn--xs tx-cat-btn"
                :style="catStyle(tx)"
                @click.stop="openCategoryPicker(tx, $event)"
                :title="catName(tx) ?? 'Assign category'"
              >
                <span
                  class="tx-cat-btn__dot"
                  :style="{ background: catColor(tx) }"
                />
                {{ catName(tx) ?? 'Categorize' }}
              </button>

              <button class="btn btn--ghost btn--icon-sm" @click="openEdit(tx)" aria-label="Edit">
                <Pencil :size="12" />
              </button>
              <button class="btn btn--ghost btn--icon-sm btn--danger" @click="confirmDelete(tx)" aria-label="Delete">
                <Trash2 :size="12" />
              </button>
            </div>
          </div>
        </div>
      </section>

      <!-- Load more -->
      <div v-if="txStore.transactions.length < txStore.total" class="tx-loadmore">
        <button class="btn btn--ghost" :disabled="txStore.loading" @click="loadMore">
          <Loader2 v-if="txStore.loading" :size="14" class="spin" />
          Load more
          <span class="tx-loadmore__count">
            ({{ txStore.total - txStore.transactions.length }} remaining)
          </span>
        </button>
      </div>
    </div>

    <!-- Category picker popover -->
    <CategoryPicker
      v-if="pickerTx"
      :categories="categoriesStore.categories"
      :model-value="pickerTx.categoryId ?? null"
      :anchor-x="pickerPos.x"
      :anchor-y="pickerPos.y"
      @update:model-value="assignCategory($event)"
      @close="pickerTx = null"
    />

    <!-- Delete confirm -->
    <Teleport to="body">
      <div v-if="deleteTarget" class="modal-backdrop" @click.self="deleteTarget = null">
        <div class="modal modal--sm" role="alertdialog">
          <div class="modal__header">
            <h2 class="modal__title">Delete transaction?</h2>
          </div>
          <div class="modal__body">
            <p class="text-body" style="color:var(--color-text-secondary)">
              <strong style="color:var(--color-text-primary)">{{ deleteTarget.payeeNormalized || deleteTarget.payee }}</strong>
              · {{ formatMoney(deleteTarget.amountCents) }} will be permanently removed.
            </p>
            <p v-if="deleteError" class="modal__error">{{ deleteError }}</p>
            <div class="modal__footer">
              <button class="btn btn--ghost" @click="deleteTarget = null">Cancel</button>
              <button class="btn btn--danger" :disabled="deleteBusy" @click="doDelete">
                <Loader2 v-if="deleteBusy" :size="14" class="spin" />
                Delete
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Add / Edit modal -->
    <TransactionModal
      v-if="showModal"
      :transaction="editTarget ?? undefined"
      :accounts="accountsStore.accounts"
      :categories="categoriesStore.categories"
      :default-account-id="filterAccountId"
      @close="closeModal"
      @saved="onSaved"
    />

    <!-- Import modal -->
    <ImportModal
      v-if="showImportModal"
      :accounts="accountsStore.accounts"
      @close="showImportModal = false"
      @imported="onImported"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import {
  Plus, Upload, Search, FilterX, Pencil, Trash2, Loader2, Receipt, X
} from 'lucide-vue-next'
import TransactionRow  from '../components/ui/TransactionRow.vue'
import CategoryPicker  from '../components/ui/CategoryPicker.vue'
import TransactionModal from '../components/ui/TransactionModal.vue'
import ImportModal      from '../components/ui/ImportModal.vue'
import { useTransactionsStore } from '../stores/transactions'
import { useAccountsStore }     from '../stores/accounts'
import { useCategoriesStore }   from '../stores/categories'
import type { Transaction }     from '../stores/transactions'

const txStore         = useTransactionsStore()
const accountsStore   = useAccountsStore()
const categoriesStore = useCategoriesStore()

// ── Filters (local mirrors → push to store on change) ─────────────────────────
const searchDraft     = ref(txStore.filter.search ?? '')
const filterAccountId  = ref<number | undefined>(txStore.filter.accountId)
const filterCategoryId = ref<number | undefined>(txStore.filter.categoryId)
const filterDateFrom   = ref(txStore.filter.dateFrom ?? '')
const filterDateTo     = ref(txStore.filter.dateTo   ?? '')

let searchTimer: ReturnType<typeof setTimeout> | null = null

function onSearchInput() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(applyFilters, 350)
}

function clearSearch() {
  searchDraft.value = ''
  applyFilters()
}

function applyFilters() {
  txStore.filter.search     = searchDraft.value
  txStore.filter.accountId  = filterAccountId.value
  txStore.filter.categoryId = filterCategoryId.value
  txStore.filter.dateFrom   = filterDateFrom.value
  txStore.filter.dateTo     = filterDateTo.value
  txStore.filter.offset     = 0
  txStore.fetch()
}

function clearAllFilters() {
  searchDraft.value     = ''
  filterAccountId.value  = undefined
  filterCategoryId.value = undefined
  filterDateFrom.value   = ''
  filterDateTo.value     = ''
  applyFilters()
}

const hasActiveFilters = computed(() =>
  !!searchDraft.value ||
  filterAccountId.value  != null ||
  filterCategoryId.value != null ||
  !!filterDateFrom.value ||
  !!filterDateTo.value
)

// ── Load more ─────────────────────────────────────────────────────────────────
function loadMore() {
  txStore.filter.offset = txStore.transactions.length
  txStore.fetch()
}

// ── Grouping by date ──────────────────────────────────────────────────────────
function txDate(tx: Transaction): string {
  const d = tx.date
  if (typeof d === 'string') return d.substring(0, 10)
  if (d instanceof Date) return d.toISOString().substring(0, 10)
  return String(d).substring(0, 10)
}

interface TxGroup {
  date:         string
  transactions: Transaction[]
  net:          number
}

const groups = computed<TxGroup[]>(() => {
  const map = new Map<string, Transaction[]>()
  for (const tx of txStore.transactions) {
    const key = txDate(tx)
    ;(map.get(key) ?? map.set(key, []).get(key)!).push(tx)
  }
  return Array.from(map.entries()).map(([date, txs]) => ({
    date,
    transactions: txs,
    net: txs.reduce((s, t) => s + t.amountCents, 0),
  }))
})

// ── Formatting ────────────────────────────────────────────────────────────────
function formatMoney(cents: number) {
  const abs = Math.abs(cents)
  return (cents < 0 ? '-$' : '$') + (abs / 100).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function formatGroupDate(iso: string) {
  const d     = new Date(iso + 'T12:00:00') // noon avoids DST shift
  const today = new Date()
  const yesterday = new Date(today); yesterday.setDate(today.getDate() - 1)
  if (iso === today.toISOString().substring(0, 10))     return 'Today'
  if (iso === yesterday.toISOString().substring(0, 10)) return 'Yesterday'
  return d.toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' })
}

// ── Category helpers ──────────────────────────────────────────────────────────
function cat(tx: Transaction) {
  return tx.categoryId ? categoriesStore.findById(tx.categoryId) : null
}
function catName(tx: Transaction) { return cat(tx)?.name ?? null }
function catColor(tx: Transaction) { return cat(tx)?.color ?? '#5A6B60' }
function catStyle(tx: Transaction) {
  const c = catColor(tx)
  return { '--cat-color': c }
}

// ── Category picker ───────────────────────────────────────────────────────────
const pickerTx  = ref<Transaction | null>(null)
const pickerPos = ref({ x: 0, y: 0 })

function openCategoryPicker(tx: Transaction, event: MouseEvent) {
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  pickerPos.value = { x: rect.left, y: rect.bottom + 4 }
  pickerTx.value  = tx
}

async function assignCategory(categoryId: number | null) {
  if (!pickerTx.value) return
  const updated = { ...pickerTx.value, categoryId: categoryId ?? undefined } as Transaction
  await txStore.update(updated)
  pickerTx.value = null
}

// ── Import modal ──────────────────────────────────────────────────────────────
const showImportModal = ref(false)

function onImported() {
  txStore.resetFilter()
  txStore.fetch()
}

// ── Create / Edit modal ───────────────────────────────────────────────────────
const showModal  = ref(false)
const editTarget = ref<Transaction | null>(null)

function openCreate() { editTarget.value = null; showModal.value = true }
function openEdit(tx: Transaction) { editTarget.value = { ...tx } as Transaction; showModal.value = true }
function closeModal() { showModal.value = false; editTarget.value = null }

async function onSaved(draft: Partial<Transaction>) {
  if (editTarget.value) {
    await txStore.update({ ...editTarget.value, ...draft } as Transaction)
  } else {
    await txStore.create(draft as Omit<Transaction, 'id' | 'createdAt' | 'updatedAt'>)
  }
  closeModal()
}

// ── Delete ────────────────────────────────────────────────────────────────────
const deleteTarget = ref<Transaction | null>(null)
const deleteBusy   = ref(false)
const deleteError  = ref<string | null>(null)

function confirmDelete(tx: Transaction) {
  deleteError.value  = null
  deleteTarget.value = tx
}

async function doDelete() {
  if (!deleteTarget.value) return
  deleteBusy.value  = true
  deleteError.value = null
  try {
    await txStore.remove(deleteTarget.value.id)
    deleteTarget.value = null
  } catch (e: any) {
    deleteError.value = e?.message ?? 'Failed to delete'
  } finally {
    deleteBusy.value = false
  }
}

// ── Init ──────────────────────────────────────────────────────────────────────
onMounted(() => {
  txStore.resetFilter()
  txStore.fetch()
})
</script>

<style scoped>
/* ── Filter bar ── */
.filter-bar {
  display: flex; align-items: center; gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  flex-wrap: wrap;
  margin-bottom: var(--space-4);
}

.filter-bar__search {
  display: flex; align-items: center; gap: var(--space-2);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: 0 var(--space-2);
  height: 34px;
  flex: 1; min-width: 160px;
}

.filter-bar__search-icon { color: var(--color-text-tertiary); flex-shrink: 0; }

.filter-bar__search-input {
  flex: 1; background: transparent; border: none; outline: none;
  font: var(--text-body-sm); color: var(--color-text-primary);
  min-width: 0;
}
.filter-bar__search-input::placeholder { color: var(--color-text-tertiary); }

.filter-bar__clear-btn {
  display: flex; align-items: center; justify-content: center;
  width: 18px; height: 18px; border-radius: 50%;
  border: none; cursor: pointer;
  background: var(--color-surface-2); color: var(--color-text-tertiary);
  flex-shrink: 0;
}
.filter-bar__clear-btn:hover { background: var(--color-border); color: var(--color-text-primary); }

.filter-bar__select {
  height: 34px; padding: 0 28px 0 var(--space-3);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text-primary);
  font: var(--text-body-sm);
  cursor: pointer; outline: none; appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='11' height='11' viewBox='0 0 24 24' fill='none' stroke='%235A6B60' stroke-width='2.5' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E");
  background-repeat: no-repeat; background-position: right var(--space-2) center;
  white-space: nowrap;
}
.filter-bar__select:focus { border-color: var(--color-primary); }

.filter-bar__date {
  height: 34px; padding: 0 var(--space-2);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text-primary);
  font: var(--text-body-sm); cursor: pointer; outline: none;
}
.filter-bar__date:focus { border-color: var(--color-primary); }

.filter-bar__date-sep {
  color: var(--color-text-tertiary); font: var(--text-body-sm);
  flex-shrink: 0;
}

.filter-badge {
  display: inline-block;
  font-size: 10px; font-weight: 600; letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--color-primary);
  background: rgba(26,138,97,0.12);
  border-radius: var(--radius-sm);
  padding: 1px 5px;
  margin-left: var(--space-2);
}

/* ── Transaction groups ── */
.tx-groups {
  display: flex; flex-direction: column; gap: var(--space-5);
  padding-bottom: var(--space-8);
}

.tx-group__header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 0 var(--space-1);
  margin-bottom: var(--space-2);
}

.tx-group__date {
  font-size: 12px; font-weight: 700; letter-spacing: 0.06em;
  text-transform: uppercase; color: var(--color-text-tertiary);
}

.tx-group__total {
  font: var(--text-body-sm); font-weight: 600;
  color: var(--color-text-secondary);
}

/* ── Transaction card rows ── */
.tx-card { padding: 0; overflow: hidden; }

.tx-card__row {
  display: flex; align-items: center;
  border-bottom: 1px solid var(--color-border);
  position: relative;
  padding-right: var(--space-2);
}
.tx-card__row--last { border-bottom: none; }

/* TransactionRow fills the row */
.tx-card__row :deep(.tx-row) {
  flex: 1;
  border-radius: 0;
  min-width: 0;
}

/* Overlay reveals on hover */
.tx-card__overlay {
  display: flex; align-items: center; gap: var(--space-1);
  opacity: 0;
  transition: opacity var(--duration-fast) var(--ease-out);
  flex-shrink: 0;
}
.tx-card__row:hover .tx-card__overlay { opacity: 1; }

/* Category quick-assign button */
.tx-cat-btn {
  display: flex; align-items: center; gap: 5px;
  height: 26px; padding: 0 var(--space-2);
  font-size: 11px; font-weight: 500;
  border-radius: var(--radius-sm);
  white-space: nowrap;
  color: var(--cat-color, var(--color-text-secondary));
}
.tx-cat-btn:hover { background: rgba(0,0,0,0.06); }

.tx-cat-btn__dot {
  width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0;
}

/* Small icon buttons */
.btn--icon-sm {
  width: 28px; height: 28px; padding: 0;
  display: flex; align-items: center; justify-content: center;
}

.btn--danger { color: var(--color-expense); }
.btn--danger:hover { background: rgba(220,38,38,0.08); }

/* Small variant */
.btn--xs {
  padding: 0 var(--space-2);
  font: var(--text-body-sm);
}

/* ── Empty state ── */
.tx-empty {
  display: flex; flex-direction: column; align-items: center;
  text-align: center; gap: var(--space-4);
  padding: var(--space-12) var(--space-8);
}

.tx-empty__icon {
  width: 56px; height: 56px; border-radius: var(--radius-lg);
  background: var(--color-surface-2);
  display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary);
}

.tx-empty__title { font: var(--text-heading); color: var(--color-text-primary); }
.tx-empty__sub {
  font: var(--text-body); color: var(--color-text-secondary);
  max-width: 340px; line-height: 1.6;
}

/* ── Skeleton ── */
.tx-skeleton { display: flex; flex-direction: column; gap: var(--space-2); margin-top: var(--space-2); }
.skeleton-row {
  height: 56px; border-radius: var(--radius-md);
  background: linear-gradient(90deg, var(--color-surface-2) 25%, var(--color-surface-1) 50%, var(--color-surface-2) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite;
}
@keyframes shimmer { to { background-position: -200% 0 } }

/* ── Load more ── */
.tx-loadmore {
  display: flex; justify-content: center;
  padding: var(--space-4) 0;
}

.tx-loadmore__count { color: var(--color-text-tertiary); margin-left: var(--space-1); }

/* ── Delete modal ── */
.modal-backdrop {
  position: fixed; inset: 0; z-index: 200;
  background: rgba(0,0,0,0.5); backdrop-filter: blur(4px);
  display: flex; align-items: center; justify-content: center;
  padding: var(--space-4);
  animation: fade-in var(--duration-base) var(--ease-out);
}
@keyframes fade-in { from { opacity: 0 } to { opacity: 1 } }

.modal {
  background: var(--color-surface-1);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  width: 100%; max-width: 420px;
  box-shadow: 0 24px 64px rgba(0,0,0,0.3);
  animation: slide-up var(--duration-base) var(--ease-out);
}
@keyframes slide-up {
  from { opacity: 0; transform: translateY(12px) }
  to   { opacity: 1; transform: translateY(0) }
}

.modal--sm { max-width: 380px; }

.modal__header {
  display: flex; align-items: center;
  padding: var(--space-5) var(--space-5) 0;
}

.modal__title { font-size: 17px; font-weight: 600; color: var(--color-text-primary); }

.modal__body {
  padding: var(--space-5);
  display: flex; flex-direction: column; gap: var(--space-4);
}

.modal__error {
  background: rgba(220,38,38,0.08);
  border: 1px solid rgba(220,38,38,0.25);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  font: var(--text-body-sm); color: var(--color-expense);
}

.modal__footer {
  display: flex; justify-content: flex-end; gap: var(--space-3);
  padding-top: var(--space-2);
  border-top: 1px solid var(--color-border);
}

.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
