<template>
  <div class="page">

    <!-- Header -->
    <div class="page-header flex items-center justify-between">
      <div>
        <h1 class="page-title">Budgets</h1>
        <p class="page-subtitle">{{ monthLabel }}</p>
      </div>
      <div class="flex items-center gap-2">
        <!-- Month nav -->
        <div class="month-nav">
          <button class="btn btn--ghost btn--icon-sm" @click="store.prevMonth()" aria-label="Previous month">
            <ChevronLeft :size="16" />
          </button>
          <span class="month-nav__label">{{ monthLabel }}</span>
          <button
            class="btn btn--ghost btn--icon-sm"
            @click="store.nextMonth()"
            :disabled="isFutureMonth"
            aria-label="Next month"
          >
            <ChevronRight :size="16" />
          </button>
        </div>
        <button class="btn btn--primary btn--sm" @click="openCreate">
          <Plus :size="14" />
          Add budget
        </button>
      </div>
    </div>

    <!-- Summary strip -->
    <div v-if="store.page" class="summary-strip card">
      <!-- Overall spend vs budget bar -->
      <div class="summary-strip__left">
        <p class="text-label" style="color:var(--color-text-secondary)">TOTAL SPENT</p>
        <div class="summary-strip__amounts">
          <span class="summary-strip__spent tabular-nums">{{ formatMoney(store.page.totalSpentCents) }}</span>
          <span class="summary-strip__of">of</span>
          <span class="summary-strip__budget tabular-nums">{{ formatMoney(store.page.totalBudgetCents) }}</span>
        </div>
        <div class="progress-bar">
          <div
            class="progress-bar__fill"
            :style="{
              width: totalBarWidth,
              background: totalBarColor,
            }"
          />
        </div>
      </div>

      <div class="summary-strip__stats">
        <div class="summary-stat">
          <span class="summary-stat__value tabular-nums" :class="remainingCents >= 0 ? 'amount--positive' : 'amount--negative'">
            {{ remainingCents >= 0 ? '+' : '' }}{{ formatMoney(remainingCents) }}
          </span>
          <span class="summary-stat__label">{{ remainingCents >= 0 ? 'remaining' : 'over budget' }}</span>
        </div>
        <div class="summary-stat">
          <span class="summary-stat__value tabular-nums">{{ store.page.lines?.length ?? 0 }}</span>
          <span class="summary-stat__label">categories budgeted</span>
        </div>
      </div>
    </div>

    <!-- Loading skeleton -->
    <div v-if="store.loading && !store.page" class="bud-skeleton">
      <div v-for="i in 5" :key="i" class="skeleton-row" />
    </div>

    <!-- Empty state -->
    <div v-else-if="!store.loading && !hasLines" class="bud-empty card">
      <div class="bud-empty__icon">
        <Target :size="28" />
      </div>
      <h2 class="bud-empty__title">No budgets yet</h2>
      <p class="bud-empty__sub">Set monthly spending targets per category to stay on track.</p>
      <button class="btn btn--primary" @click="openCreate">
        <Plus :size="14" />
        Add your first budget
      </button>
    </div>

    <!-- Budget lines -->
    <div v-else class="bud-groups">

      <!-- Budgeted categories -->
      <section v-if="store.page?.lines?.length">
        <div class="bud-section-header">
          <span class="bud-section-label">Budgeted</span>
        </div>
        <div class="card bud-card">
          <div
            v-for="(line, idx) in store.page.lines"
            :key="line.id"
            class="bud-row"
            :class="{ 'bud-row--last': idx === (store.page?.lines?.length ?? 0) - 1 }"
          >
            <!-- Icon -->
            <div class="bud-row__icon" :style="{ background: `${line.categoryColor}18`, color: line.categoryColor }">
              <component :is="iconFor(line.categoryIcon)" :size="14" />
            </div>

            <!-- Info -->
            <div class="bud-row__info">
              <div class="bud-row__name-row">
                <span class="bud-row__name">{{ line.categoryName }}</span>
                <span v-if="line.rollsOver" class="bud-row__badge">Rollover</span>
                <span
                  class="bud-row__overage"
                  v-if="line.spentCents > line.budgetCents"
                >+{{ formatMoney(line.spentCents - line.budgetCents) }} over</span>
              </div>
              <div class="bud-row__bar-row">
                <div class="progress-bar bud-row__bar">
                  <div
                    class="progress-bar__fill"
                    :style="{
                      width: lineBarWidth(line),
                      background: lineBarColor(line),
                    }"
                  />
                </div>
                <span class="bud-row__amounts tabular-nums">
                  {{ formatMoney(line.spentCents) }}
                  <span style="color:var(--color-text-tertiary)"> / {{ formatMoney(line.budgetCents) }}</span>
                </span>
              </div>
            </div>

            <!-- Actions -->
            <div class="bud-row__actions">
              <button class="btn btn--ghost btn--icon-sm" @click="openEdit(line)" aria-label="Edit">
                <Pencil :size="12" />
              </button>
              <button class="btn btn--ghost btn--icon-sm btn--danger" @click="confirmDelete(line)" aria-label="Delete">
                <Trash2 :size="12" />
              </button>
            </div>
          </div>
        </div>
      </section>

      <!-- Unbudgeted spending -->
      <section v-if="store.page?.unbudgeted?.length">
        <div class="bud-section-header">
          <span class="bud-section-label">Unbudgeted spending</span>
          <span class="bud-section-hint">These categories have expenses but no budget set.</span>
        </div>
        <div class="card bud-card">
          <div
            v-for="(u, idx) in store.page.unbudgeted"
            :key="u.categoryId"
            class="bud-row bud-row--unbudgeted"
            :class="{ 'bud-row--last': idx === (store.page?.unbudgeted?.length ?? 0) - 1 }"
          >
            <div class="bud-row__icon" :style="{ background: `${u.categoryColor}18`, color: u.categoryColor }">
              <component :is="iconFor(u.categoryIcon)" :size="14" />
            </div>
            <div class="bud-row__info">
              <span class="bud-row__name">{{ u.categoryName }}</span>
            </div>
            <span class="bud-row__unbudgeted-amount tabular-nums">{{ formatMoney(u.spentCents) }}</span>
            <button
              class="btn btn--ghost btn--xs"
              @click="quickAddBudget(u)"
              title="Add a budget for this category"
            >
              <Plus :size="12" />
              Budget it
            </button>
          </div>
        </div>
      </section>
    </div>

    <!-- Delete confirm -->
    <Teleport to="body">
      <div v-if="deleteTarget" class="modal-backdrop" @click.self="deleteTarget = null">
        <div class="modal modal--sm" role="alertdialog" aria-labelledby="bud-delete-title">
          <div class="modal__header">
            <h2 id="bud-delete-title" class="modal__title">Remove budget?</h2>
          </div>
          <div class="modal__body">
            <p class="text-body" style="color:var(--color-text-secondary)">
              The budget for <strong style="color:var(--color-text-primary)">{{ deleteTarget.categoryName }}</strong>
              will be removed. Your transactions won't be affected.
            </p>
            <p v-if="deleteError" class="modal__error">{{ deleteError }}</p>
            <div class="modal__footer">
              <button class="btn btn--ghost" @click="deleteTarget = null">Cancel</button>
              <button class="btn btn--danger" :disabled="deleteBusy" @click="doDelete">
                <Loader2 v-if="deleteBusy" :size="14" class="spin" />
                Remove budget
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Add / Edit modal -->
    <BudgetModal
      v-if="showModal"
      :line="editTarget ?? undefined"
      :categories="categoriesStore.categories"
      :budgeted-ids="budgetedIds"
      @close="closeModal"
      @saved="onSaved"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, markRaw } from 'vue'
import {
  Plus, ChevronLeft, ChevronRight, Pencil, Trash2, Loader2, Target,
  Home, Apple, Tv, Banknote, Car, Heart, Ticket, ShoppingBag,
  PiggyBank, TrendingUp, ArrowLeftRight, Tag,
} from 'lucide-vue-next'
import BudgetModal from '../components/ui/BudgetModal.vue'
import { useBudgetsStore }     from '../stores/budgets'
import { useCategoriesStore }  from '../stores/categories'
import type { Budget, BudgetLine, UnbudgetedLine } from '../stores/budgets'

const store            = useBudgetsStore()
const categoriesStore  = useCategoriesStore()

// ── Icon map (mirrors TransactionRow) ────────────────────────────────────────
const iconMap: Record<string, any> = {
  'home':             markRaw(Home),
  'utensils':         markRaw(Apple),
  'tv':               markRaw(Tv),
  'banknote':         markRaw(Banknote),
  'car':              markRaw(Car),
  'heart-pulse':      markRaw(Heart),
  'ticket':           markRaw(Ticket),
  'shopping-bag':     markRaw(ShoppingBag),
  'piggy-bank':       markRaw(PiggyBank),
  'trending-up':      markRaw(TrendingUp),
  'arrow-left-right': markRaw(ArrowLeftRight),
  'tag':              markRaw(Tag),
}
function iconFor(icon: string) { return iconMap[icon] ?? markRaw(Tag) }

// ── Month label ───────────────────────────────────────────────────────────────
const monthLabel = computed(() => {
  const d = new Date(store.year, store.month - 1, 1)
  return d.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })
})

const isFutureMonth = computed(() => {
  const now = new Date()
  return store.year > now.getFullYear() ||
    (store.year === now.getFullYear() && store.month > now.getMonth() + 1)
})

// ── Summary bar ───────────────────────────────────────────────────────────────
const remainingCents = computed(() =>
  (store.page?.totalBudgetCents ?? 0) - (store.page?.totalSpentCents ?? 0)
)

const totalRatio = computed(() => {
  const bud = store.page?.totalBudgetCents ?? 0
  const spt = store.page?.totalSpentCents  ?? 0
  if (bud === 0) return spt > 0 ? 1 : 0
  return Math.min(spt / bud, 1)
})

const totalBarWidth = computed(() => `${totalRatio.value * 100}%`)
const totalBarColor = computed(() => barColorFor(totalRatio.value))

// ── Line bar helpers ──────────────────────────────────────────────────────────
function lineRatio(line: BudgetLine) {
  if (line.budgetCents === 0) return line.spentCents > 0 ? 1 : 0
  return Math.min(line.spentCents / line.budgetCents, 1)
}
function lineBarWidth(line: BudgetLine) { return `${lineRatio(line) * 100}%` }
function lineBarColor(line: BudgetLine) { return barColorFor(lineRatio(line)) }

function barColorFor(ratio: number) {
  if (ratio < 0.75) return 'var(--color-primary)'
  if (ratio < 0.90) return 'var(--color-gold-500)'
  if (ratio < 1.0)  return 'var(--color-warning)'
  return 'var(--color-expense)'
}

// ── Formatting ────────────────────────────────────────────────────────────────
function formatMoney(cents: number) {
  const abs = Math.abs(cents)
  return (cents < 0 ? '-$' : '$') +
    (abs / 100).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

// ── Has content guard ─────────────────────────────────────────────────────────
const hasLines = computed(() =>
  (store.page?.lines?.length ?? 0) > 0 || (store.page?.unbudgeted?.length ?? 0) > 0
)

const budgetedIds = computed(() =>
  store.page?.lines?.map(l => l.categoryId) ?? []
)

// ── Create / Edit ─────────────────────────────────────────────────────────────
const showModal  = ref(false)
const editTarget = ref<BudgetLine | null>(null)
const prefillCat = ref<number | null>(null)

function openCreate() {
  prefillCat.value  = null
  editTarget.value  = null
  showModal.value   = true
}

function openEdit(line: BudgetLine) {
  editTarget.value = { ...line } as BudgetLine
  showModal.value  = true
}

function closeModal() {
  showModal.value  = false
  editTarget.value = null
  prefillCat.value = null
}

function quickAddBudget(u: UnbudgetedLine) {
  // Pre-select the category in the create modal
  prefillCat.value = u.categoryId
  editTarget.value = null
  showModal.value  = true
}

async function onSaved(payload: { categoryId: number; amountCents: number; rollsOver: boolean }) {
  const today = new Date().toISOString().substring(0, 10)

  if (editTarget.value) {
    await store.update({
      id:          editTarget.value.id,
      categoryId:  payload.categoryId,
      amountCents: payload.amountCents,
      period:      'monthly',
      rollsOver:   payload.rollsOver,
      startDate:   today,
      endDate:     undefined,
    } as Budget)
  } else {
    await store.create({
      categoryId:  payload.categoryId,
      amountCents: payload.amountCents,
      period:      'monthly',
      rollsOver:   payload.rollsOver,
      startDate:   today,
      endDate:     undefined,
    })
  }
  closeModal()
}

// ── Delete ────────────────────────────────────────────────────────────────────
const deleteTarget = ref<BudgetLine | null>(null)
const deleteBusy   = ref(false)
const deleteError  = ref<string | null>(null)

function confirmDelete(line: BudgetLine) {
  deleteError.value  = null
  deleteTarget.value = line
}

async function doDelete() {
  if (!deleteTarget.value) return
  deleteBusy.value  = true
  deleteError.value = null
  try {
    await store.remove(deleteTarget.value.id)
    deleteTarget.value = null
  } catch (e: any) {
    deleteError.value = e?.message ?? 'Failed to remove budget'
  } finally {
    deleteBusy.value = false
  }
}

// ── Init ──────────────────────────────────────────────────────────────────────
onMounted(() => store.fetch())
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
  min-width: 110px; text-align: center;
}

.btn--icon-sm {
  width: 28px; height: 28px; padding: 0;
  display: flex; align-items: center; justify-content: center;
}

/* ── Summary strip ── */
.summary-strip {
  display: flex; align-items: center;
  gap: var(--space-8); padding: var(--space-4) var(--space-5);
  margin-bottom: var(--space-5);
}

.summary-strip__left { flex: 1; }

.summary-strip__amounts {
  display: flex; align-items: baseline; gap: var(--space-2);
  margin: var(--space-1) 0 var(--space-2);
}

.summary-strip__spent {
  font-size: 24px; font-weight: 700;
  font-family: var(--font-display);
  color: var(--color-text-primary);
}

.summary-strip__of { font: var(--text-body); color: var(--color-text-tertiary); }

.summary-strip__budget {
  font: var(--text-body); font-weight: 600;
  color: var(--color-text-secondary);
}

.summary-strip__stats {
  display: flex; gap: var(--space-8); flex-shrink: 0;
}

.summary-stat {
  display: flex; flex-direction: column; align-items: center; gap: 2px;
}

.summary-stat__value {
  font-size: 20px; font-weight: 700;
  font-family: var(--font-display);
  color: var(--color-text-primary);
}

.summary-stat__label {
  font: var(--text-body-sm); color: var(--color-text-tertiary);
  white-space: nowrap;
}

/* ── Progress bar (shared) ── */
.progress-bar {
  height: 6px; border-radius: var(--radius-full);
  background: var(--color-surface-2); overflow: hidden;
}

.progress-bar__fill {
  height: 100%; border-radius: var(--radius-full);
  transition: width 600ms var(--ease-out), background-color var(--duration-base) var(--ease-out);
}

/* ── Sections ── */
.bud-groups { display: flex; flex-direction: column; gap: var(--space-5); padding-bottom: var(--space-8); }

.bud-section-header {
  display: flex; align-items: baseline; gap: var(--space-3);
  padding: 0 var(--space-1); margin-bottom: var(--space-3);
}

.bud-section-label {
  font-size: 11px; font-weight: 700; letter-spacing: 0.08em;
  text-transform: uppercase; color: var(--color-text-tertiary);
}

.bud-section-hint {
  font: var(--text-body-sm); color: var(--color-text-tertiary);
}

/* ── Budget rows ── */
.bud-card { padding: 0; overflow: hidden; }

.bud-row {
  display: flex; align-items: center; gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--color-border);
  transition: background var(--duration-fast) var(--ease-out);
}
.bud-row--last { border-bottom: none; }
.bud-row:hover { background: var(--color-surface-2); }

.bud-row__icon {
  width: 32px; height: 32px; border-radius: var(--radius-md);
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}

.bud-row__info { flex: 1; min-width: 0; }

.bud-row__name-row {
  display: flex; align-items: center; gap: var(--space-2);
  margin-bottom: var(--space-1);
}

.bud-row__name {
  font: var(--text-body); font-weight: 500;
  color: var(--color-text-primary);
}

.bud-row__badge {
  font-size: 10px; font-weight: 600; letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--color-primary);
  background: rgba(26,138,97,0.12);
  border-radius: var(--radius-sm);
  padding: 1px 5px;
}

.bud-row__overage {
  font-size: 11px; font-weight: 600;
  color: var(--color-expense);
  margin-left: auto;
}

.bud-row__bar-row {
  display: flex; align-items: center; gap: var(--space-3);
}

.bud-row__bar { flex: 1; }

.bud-row__amounts {
  font: var(--text-body-sm); font-weight: 600;
  color: var(--color-text-primary);
  white-space: nowrap; flex-shrink: 0;
}

/* Actions (reveal on hover) */
.bud-row__actions {
  display: flex; gap: var(--space-1);
  opacity: 0;
  transition: opacity var(--duration-fast) var(--ease-out);
  flex-shrink: 0;
}
.bud-row:hover .bud-row__actions { opacity: 1; }

.btn--danger { color: var(--color-expense); }
.btn--danger:hover { background: rgba(220,38,38,0.08); }

/* Unbudgeted rows */
.bud-row--unbudgeted .bud-row__actions { opacity: 1; }

.bud-row__unbudgeted-amount {
  font: var(--text-body); font-weight: 600;
  color: var(--color-text-primary);
  margin-left: auto; flex-shrink: 0;
}

.btn--xs { height: 26px; padding: 0 var(--space-2); font: var(--text-body-sm); font-weight: 500; }

/* ── Empty state ── */
.bud-empty {
  display: flex; flex-direction: column; align-items: center;
  text-align: center; gap: var(--space-4);
  padding: var(--space-12) var(--space-8);
}

.bud-empty__icon {
  width: 56px; height: 56px; border-radius: var(--radius-lg);
  background: var(--color-surface-2);
  display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary);
}

.bud-empty__title { font: var(--text-heading); color: var(--color-text-primary); }
.bud-empty__sub { font: var(--text-body); color: var(--color-text-secondary); max-width: 340px; line-height: 1.6; }

/* ── Skeleton ── */
.bud-skeleton { display: flex; flex-direction: column; gap: var(--space-2); margin-top: var(--space-4); }
.skeleton-row {
  height: 72px; border-radius: var(--radius-md);
  background: linear-gradient(90deg, var(--color-surface-2) 25%, var(--color-surface-1) 50%, var(--color-surface-2) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite;
}
@keyframes shimmer { to { background-position: -200% 0 } }

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

/* Amount color helpers */
.amount--positive { color: var(--color-income); }
.amount--negative { color: var(--color-expense); }
</style>
