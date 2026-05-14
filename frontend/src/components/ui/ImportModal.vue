<template>
  <Teleport to="body">
    <div class="modal-backdrop" @click.self="maybeClose">
      <div ref="modalRef" class="modal import-modal" role="dialog" aria-modal="true" aria-label="Import transactions">

        <!-- ── Step 1: Choose source ─────────────────────────────────────── -->
        <template v-if="step === 'pick'">
          <div class="modal__header">
            <h2 class="modal__title">Import transactions</h2>
            <button class="btn btn--ghost btn--icon" @click="$emit('close')" aria-label="Close">
              <X :size="16" />
            </button>
          </div>

          <div class="modal__body">
            <p class="import-help">
              Choose your export file type. Kino will detect which transactions are
              new and skip any it has already imported.
            </p>

            <!-- Account selector -->
            <div class="field">
              <label class="field__label" for="imp-account">Import into account</label>
              <select id="imp-account" v-model="accountId" class="field__input field__select" required>
                <option value="" disabled>Select account…</option>
                <option v-for="acc in accounts" :key="acc.id" :value="acc.id">{{ acc.name }}</option>
              </select>
            </div>

            <!-- File type cards -->
            <div class="import-types">
              <button
                class="import-type"
                :class="{ 'import-type--active': format === 'csv' }"
                @click="format = 'csv'"
              >
                <div class="import-type__icon"><FileText :size="22" /></div>
                <div>
                  <p class="import-type__name">CSV</p>
                  <p class="import-type__desc">Bank export, spreadsheet</p>
                </div>
              </button>
              <button
                class="import-type"
                :class="{ 'import-type--active': format === 'ofx' }"
                @click="format = 'ofx'"
              >
                <div class="import-type__icon"><Landmark :size="22" /></div>
                <div>
                  <p class="import-type__name">OFX / QFX</p>
                  <p class="import-type__desc">Quicken, most banks</p>
                </div>
              </button>
            </div>

            <p v-if="pickError" class="modal__error">{{ pickError }}</p>
          </div>

          <div class="modal__footer">
            <button class="btn btn--ghost" @click="$emit('close')">Cancel</button>
            <button
              class="btn btn--primary"
              :disabled="!accountId || !format || importing"
              @click="runImport"
            >
              <Loader2 v-if="importing" :size="14" class="spin" />
              Choose file…
            </button>
          </div>
        </template>

        <!-- ── Step 2: Import result summary ──────────────────────────────── -->
        <template v-else-if="step === 'result'">
          <div class="modal__header">
            <h2 class="modal__title">Import complete</h2>
          </div>

          <div class="modal__body">
            <div class="result-stats">
              <div class="result-stat result-stat--inserted">
                <span class="result-stat__num">{{ result!.inserted }}</span>
                <span class="result-stat__label">imported</span>
              </div>
              <div class="result-stat result-stat--skipped">
                <span class="result-stat__num">{{ result!.skipped }}</span>
                <span class="result-stat__label">skipped</span>
              </div>
              <div
                class="result-stat"
                :class="hasDupes ? 'result-stat--warn' : 'result-stat--ok'"
              >
                <span class="result-stat__num">{{ result!.possibleDupes?.length ?? 0 }}</span>
                <span class="result-stat__label">to review</span>
              </div>
            </div>

            <p class="import-help">
              <strong>{{ result!.inserted }}</strong> new transactions from
              <em>{{ result!.fileName }}</em>.
              <span v-if="result!.skipped > 0">
                {{ result!.skipped }} {{ result!.skipped === 1 ? 'was' : 'were' }} skipped
                (already imported).
              </span>
            </p>

            <div v-if="hasDupes" class="dupe-warning">
              <AlertTriangle :size="15" class="dupe-warning__icon" />
              <p>
                {{ result!.possibleDupes.length }}
                {{ result!.possibleDupes.length === 1 ? 'transaction looks' : 'transactions look' }}
                like {{ result!.possibleDupes.length === 1 ? 'a duplicate' : 'duplicates' }}
                — same date and amount as an existing entry from a different source.
                Review them below.
              </p>
            </div>

            <div v-if="hasWarnings" class="warning-list">
              <div class="warning-list__header">
                <AlertTriangle :size="15" />
                <span>
                  {{ result!.warnings.length }}
                  {{ result!.warnings.length === 1 ? 'row was' : 'rows were' }} skipped
                </span>
              </div>
              <ul class="warning-list__items">
                <li v-for="(w, i) in result!.warnings" :key="i">{{ w }}</li>
              </ul>
            </div>
          </div>

          <div class="modal__footer">
            <button class="btn btn--ghost" @click="finishAndClose">Done</button>
            <button v-if="hasDupes" class="btn btn--primary" @click="step = 'review'">
              Review {{ result!.possibleDupes.length }} possible {{ result!.possibleDupes.length === 1 ? 'duplicate' : 'duplicates' }}
              <ChevronRight :size="14" />
            </button>
          </div>
        </template>

        <!-- ── Step 3: Duplicate review ───────────────────────────────────── -->
        <template v-else-if="step === 'review'">
          <div class="modal__header">
            <h2 class="modal__title">
              Review duplicates
              <span class="modal__progress">{{ dupeIdx + 1 }} / {{ result!.possibleDupes.length }}</span>
            </h2>
          </div>

          <div class="modal__body">
            <div v-if="currentDupe" class="dupe-pair">
              <!-- Newly imported -->
              <div class="dupe-card dupe-card--new">
                <p class="dupe-card__badge">Just imported</p>
                <p class="dupe-card__payee">{{ currentDupe.newTx.payeeNormalized || currentDupe.newTx.payee }}</p>
                <p class="dupe-card__amount tabular-nums" :class="currentDupe.newTx.amountCents < 0 ? '' : 'amount--positive'">
                  {{ formatAmount(currentDupe.newTx.amountCents) }}
                </p>
                <p class="dupe-card__meta">{{ formatDate(currentDupe.newTx.date) }}</p>
                <p class="dupe-card__source">Source: {{ currentDupe.newTx.importSource }}</p>
              </div>

              <div class="dupe-arrow"><ArrowRight :size="16" /></div>

              <!-- Existing -->
              <div class="dupe-card dupe-card--existing">
                <p class="dupe-card__badge">Already in Kino</p>
                <p class="dupe-card__payee">{{ currentDupe.existingTx.payeeNormalized || currentDupe.existingTx.payee }}</p>
                <p class="dupe-card__amount tabular-nums" :class="currentDupe.existingTx.amountCents < 0 ? '' : 'amount--positive'">
                  {{ formatAmount(currentDupe.existingTx.amountCents) }}
                </p>
                <p class="dupe-card__meta">{{ formatDate(currentDupe.existingTx.date) }}</p>
                <p class="dupe-card__source">Source: {{ currentDupe.existingTx.importSource }}</p>
              </div>
            </div>

            <p v-if="resolveError" class="modal__error">{{ resolveError }}</p>

            <!-- Action buttons -->
            <div class="dupe-actions">
              <button
                class="btn btn--ghost dupe-action"
                :disabled="resolvingDupe"
                @click="resolve('keep_both')"
                title="They really are two separate transactions"
              >
                <Copy :size="13" />
                Keep both
              </button>
              <button
                class="btn btn--ghost dupe-action"
                :disabled="resolvingDupe"
                @click="resolve('delete_new')"
                title="Discard the just-imported row"
              >
                <Trash2 :size="13" />
                Delete new
              </button>
              <button
                class="btn btn--primary dupe-action"
                :disabled="resolvingDupe"
                @click="resolve('merge')"
                title="Keep existing row, update its import hash so future syncs skip this transaction"
              >
                <Loader2 v-if="resolvingDupe" :size="13" class="spin" />
                <GitMerge v-else :size="13" />
                Merge
                <span class="dupe-action__hint">recommended</span>
              </button>
            </div>

            <p class="import-help" style="margin-top: var(--space-3)">
              <strong>Merge</strong> keeps your existing entry and teaches Kino to
              skip this transaction in future syncs.
            </p>
          </div>

          <div class="modal__footer">
            <button class="btn btn--ghost" @click="step = 'result'">
              <ChevronLeft :size="14" /> Back
            </button>
            <button class="btn btn--ghost" @click="finishAndClose">Done reviewing</button>
          </div>
        </template>

      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import {
  X, FileText, Landmark, Loader2, ChevronRight, ChevronLeft,
  AlertTriangle, ArrowRight, Copy, Trash2, GitMerge,
} from 'lucide-vue-next'
import { PickAndImportCSV, PickAndImportOFX, ResolveDuplicate } from '../../../wailsjs/go/main/App'
import type { main } from '../../../wailsjs/go/models'
import type { Account } from '../../stores/accounts'
import { useFocusTrap } from './useFocusTrap'

type ImportResult = main.ImportResult

const props = defineProps<{
  accounts: Account[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'imported'): void  // triggers parent to refresh transactions
}>()

// ── Accessibility ─────────────────────────────────────────────────────────────
const modalRef = ref<HTMLElement | null>(null)

useFocusTrap(modalRef)

function onEscape(e: KeyboardEvent) {
  if (e.key === 'Escape' && step.value === 'pick') emit('close')
}
onMounted(() => window.addEventListener('keydown', onEscape))
onUnmounted(() => window.removeEventListener('keydown', onEscape))

// ── State ─────────────────────────────────────────────────────────────────────
type Step = 'pick' | 'result' | 'review'

const step      = ref<Step>('pick')
const format    = ref<'csv' | 'ofx' | ''>('')
const accountId = ref<number | ''>('')
const importing = ref(false)
const pickError = ref<string | null>(null)
const result    = ref<ImportResult | null>(null)

// Dupe review
const dupeIdx      = ref(0)
const resolvingDupe = ref(false)
const resolveError  = ref<string | null>(null)

const hasDupes    = computed(() => (result.value?.possibleDupes?.length ?? 0) > 0)
const hasWarnings = computed(() => (result.value?.warnings?.length ?? 0) > 0)
const currentDupe = computed(() => result.value?.possibleDupes?.[dupeIdx.value] ?? null)

// ── Import ────────────────────────────────────────────────────────────────────
async function runImport() {
  if (!accountId.value || !format.value) return
  pickError.value = null
  importing.value = true

  try {
    const id = accountId.value as number
    const res = format.value === 'csv'
      ? await PickAndImportCSV(id)
      : await PickAndImportOFX(id)

    // User cancelled the file picker — backend returns null/undefined
    if (!res) {
      importing.value = false
      return
    }

    result.value = res
    step.value   = 'result'
    emit('imported')
  } catch (e: any) {
    pickError.value = e?.message ?? 'Import failed'
  } finally {
    importing.value = false
  }
}

// ── Dupe resolution ───────────────────────────────────────────────────────────
async function resolve(action: string) {
  if (!currentDupe.value) return
  resolveError.value  = null
  resolvingDupe.value = true

  try {
    const { newTx, existingTx } = currentDupe.value
    // For delete_new and merge: keepID = existing, deleteID = new
    await ResolveDuplicate(action, existingTx.id, newTx.id)
    advanceDupe()
  } catch (e: any) {
    resolveError.value = e?.message ?? 'Failed to resolve'
  } finally {
    resolvingDupe.value = false
  }
}

function advanceDupe() {
  const total = result.value?.possibleDupes?.length ?? 0
  if (dupeIdx.value + 1 < total) {
    dupeIdx.value++
  } else {
    finishAndClose()
  }
}

// ── Helpers ───────────────────────────────────────────────────────────────────
function maybeClose() {
  if (step.value === 'pick') emit('close')
}

function finishAndClose() {
  emit('imported')
  emit('close')
}

function formatAmount(cents: number) {
  const abs = Math.abs(cents)
  return (cents < 0 ? '-$' : '+$') +
    (abs / 100).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function formatDate(d: any) {
  const s = typeof d === 'string' ? d : String(d)
  const dt = new Date(s.substring(0, 10) + 'T12:00:00')
  return dt.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}
</script>

<style scoped>
.modal-backdrop {
  position: fixed; inset: 0; z-index: 200;
  background: rgba(0,0,0,0.5); backdrop-filter: blur(4px);
  display: flex; align-items: center; justify-content: center;
  padding: var(--space-4);
  animation: fade-in var(--duration-base) var(--ease-out);
}
@keyframes fade-in { from { opacity: 0 } to { opacity: 1 } }

.import-modal {
  background: var(--color-surface-1);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  width: 100%; max-width: 500px;
  box-shadow: 0 24px 64px rgba(0,0,0,0.3);
  animation: slide-up var(--duration-base) var(--ease-out);
}
@keyframes slide-up {
  from { opacity: 0; transform: translateY(12px) }
  to   { opacity: 1; transform: translateY(0) }
}

.modal__header {
  display: flex; align-items: center; justify-content: space-between;
  padding: var(--space-5) var(--space-5) 0;
}
.modal__title {
  font-size: 17px; font-weight: 600; color: var(--color-text-primary);
  display: flex; align-items: center; gap: var(--space-3);
}
.modal__progress {
  font-size: 13px; font-weight: 400; color: var(--color-text-tertiary);
}
.btn--icon {
  width: 32px; height: 32px; padding: 0;
  display: flex; align-items: center; justify-content: center;
}
.modal__body {
  padding: var(--space-5);
  display: flex; flex-direction: column; gap: var(--space-4);
}
.modal__footer {
  display: flex; justify-content: flex-end; gap: var(--space-3);
  padding: var(--space-4) var(--space-5);
  border-top: 1px solid var(--color-border);
}
.modal__error {
  background: rgba(220,38,38,0.08); border: 1px solid rgba(220,38,38,0.25);
  border-radius: var(--radius-md); padding: var(--space-3);
  font: var(--text-body-sm); color: var(--color-expense);
}

/* Help text */
.import-help {
  font: var(--text-body); color: var(--color-text-secondary); line-height: 1.6;
}

/* Format picker */
.import-types { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-3); }

.import-type {
  display: flex; align-items: center; gap: var(--space-3);
  padding: var(--space-4);
  background: var(--color-surface-2);
  border: 2px solid var(--color-border);
  border-radius: var(--radius-lg);
  cursor: pointer; text-align: left;
  transition: border-color var(--duration-fast), background var(--duration-fast);
}
.import-type:hover { border-color: var(--color-primary); background: var(--color-surface-1); }
.import-type--active {
  border-color: var(--color-primary);
  background: rgba(26,138,97,0.06);
}

.import-type__icon {
  width: 40px; height: 40px; border-radius: var(--radius-md);
  background: var(--color-surface-1);
  display: flex; align-items: center; justify-content: center;
  color: var(--color-text-secondary); flex-shrink: 0;
}
.import-type--active .import-type__icon { color: var(--color-primary); }

.import-type__name { font: var(--text-body); font-weight: 600; color: var(--color-text-primary); }
.import-type__desc { font: var(--text-body-sm); color: var(--color-text-tertiary); margin-top: 2px; }

/* Fields */
.field { display: flex; flex-direction: column; gap: var(--space-2); }
.field__label {
  font-size: 12px; font-weight: 600; letter-spacing: 0.04em;
  color: var(--color-text-secondary); text-transform: uppercase;
}
.field__input {
  height: 38px; padding: 0 var(--space-3);
  background: var(--color-surface-2); border: 1px solid var(--color-border);
  border-radius: var(--radius-md); color: var(--color-text-primary);
  font: var(--text-body); outline: none; width: 100%; box-sizing: border-box;
}
.field__select {
  cursor: pointer; appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%235A6B60' stroke-width='2.5' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E");
  background-repeat: no-repeat; background-position: right var(--space-3) center;
  padding-right: 36px;
}

/* Result stats */
.result-stats { display: flex; gap: var(--space-4); }
.result-stat {
  flex: 1; padding: var(--space-4);
  border-radius: var(--radius-md);
  background: var(--color-surface-2);
  display: flex; flex-direction: column; align-items: center; gap: var(--space-1);
}
.result-stat--inserted { border: 1px solid rgba(26,138,97,0.3); }
.result-stat--skipped  { border: 1px solid var(--color-border); }
.result-stat--warn     { border: 1px solid rgba(196,148,58,0.4); background: rgba(196,148,58,0.06); }
.result-stat--ok       { border: 1px solid rgba(26,138,97,0.2); }

.result-stat__num {
  font-size: 28px; font-weight: 700;
  font-family: var(--font-display);
  color: var(--color-text-primary);
}
.result-stat--inserted .result-stat__num { color: var(--color-primary); }
.result-stat--warn     .result-stat__num { color: var(--color-accent); }

.result-stat__label {
  font: var(--text-body-sm); color: var(--color-text-tertiary);
}

/* Dupe warning banner */
.dupe-warning {
  display: flex; align-items: flex-start; gap: var(--space-3);
  background: rgba(196,148,58,0.08);
  border: 1px solid rgba(196,148,58,0.3);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  font: var(--text-body-sm); color: var(--color-text-secondary); line-height: 1.5;
}
.dupe-warning__icon { color: var(--color-accent); flex-shrink: 0; margin-top: 1px; }

/* Skipped-row warnings list */
.warning-list {
  background: rgba(196,148,58,0.06);
  border: 1px solid rgba(196,148,58,0.2);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  font: var(--text-body-sm); color: var(--color-text-secondary);
}
.warning-list__header {
  display: flex; align-items: center; gap: var(--space-2);
  color: var(--color-accent); font-weight: 600;
  margin-bottom: var(--space-2);
}
.warning-list__items {
  list-style: disc; padding-left: var(--space-5);
  display: flex; flex-direction: column; gap: 2px;
  max-height: 140px; overflow-y: auto;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px; line-height: 1.5;
}

/* Dupe pair comparison */
.dupe-pair {
  display: flex; align-items: center; gap: var(--space-3);
}

.dupe-card {
  flex: 1; border-radius: var(--radius-md);
  padding: var(--space-4);
  display: flex; flex-direction: column; gap: var(--space-1);
}
.dupe-card--new      { background: rgba(196,148,58,0.08); border: 1px solid rgba(196,148,58,0.3); }
.dupe-card--existing { background: var(--color-surface-2); border: 1px solid var(--color-border); }

.dupe-card__badge {
  font-size: 10px; font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase;
  color: var(--color-text-tertiary); margin-bottom: var(--space-1);
}
.dupe-card--new .dupe-card__badge { color: var(--color-accent); }

.dupe-card__payee {
  font: var(--text-body); font-weight: 600; color: var(--color-text-primary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.dupe-card__amount { font: var(--text-body); font-weight: 700; color: var(--color-text-primary); }
.dupe-card__meta   { font: var(--text-body-sm); color: var(--color-text-tertiary); }
.dupe-card__source {
  font-size: 10px; color: var(--color-text-tertiary);
  text-transform: uppercase; letter-spacing: 0.05em;
}

.dupe-arrow { color: var(--color-text-tertiary); flex-shrink: 0; }

/* Dupe action buttons */
.dupe-actions { display: flex; gap: var(--space-2); flex-wrap: wrap; }
.dupe-action  { flex: 1; justify-content: center; gap: var(--space-2); min-width: 0; }

.dupe-action__hint {
  font-size: 10px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em;
  background: rgba(255,255,255,0.15); border-radius: 3px; padding: 1px 4px;
}

.amount--positive { color: var(--color-income); }

.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
