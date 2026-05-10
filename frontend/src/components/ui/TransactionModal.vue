<template>
  <Teleport to="body">
    <div class="modal-backdrop" @click.self="$emit('close')">
      <div class="modal" role="dialog" :aria-label="isEdit ? 'Edit transaction' : 'Add transaction'">

        <!-- Header -->
        <div class="modal__header">
          <h2 class="modal__title">{{ isEdit ? 'Edit transaction' : 'Add transaction' }}</h2>
          <button class="btn btn--ghost btn--icon" @click="$emit('close')" aria-label="Close">
            <X :size="16" />
          </button>
        </div>

        <form class="modal__body" @submit.prevent="submit">

          <!-- Amount + type toggle -->
          <div class="field">
            <label class="field__label">Amount</label>
            <div class="amount-row">
              <div class="type-toggle">
                <button
                  type="button"
                  class="type-toggle__btn"
                  :class="{ 'type-toggle__btn--active': txType === 'expense' }"
                  @click="txType = 'expense'"
                >Expense</button>
                <button
                  type="button"
                  class="type-toggle__btn"
                  :class="{ 'type-toggle__btn--active': txType === 'income' }"
                  @click="txType = 'income'"
                >Income</button>
              </div>
              <div class="field__input-wrap" style="flex:1">
                <span class="field__prefix">$</span>
                <input
                  v-model="amountDisplay"
                  class="field__input field__input--prefixed"
                  type="number"
                  step="0.01"
                  min="0"
                  placeholder="0.00"
                  required
                />
              </div>
            </div>
          </div>

          <!-- Date -->
          <div class="field">
            <label class="field__label" for="tx-date">Date</label>
            <input
              id="tx-date"
              v-model="form.date"
              class="field__input"
              type="date"
              required
            />
          </div>

          <!-- Payee -->
          <div class="field">
            <label class="field__label" for="tx-payee">Payee</label>
            <input
              id="tx-payee"
              v-model="form.payee"
              class="field__input"
              placeholder="e.g. Whole Foods"
              required
              autocomplete="off"
            />
          </div>

          <!-- Account -->
          <div class="field">
            <label class="field__label" for="tx-account">Account</label>
            <select id="tx-account" v-model="form.accountId" class="field__input field__select" required>
              <option value="" disabled>Select account…</option>
              <option v-for="acc in accounts" :key="acc.id" :value="acc.id">{{ acc.name }}</option>
            </select>
          </div>

          <!-- Category -->
          <div class="field">
            <label class="field__label" for="tx-category">Category</label>
            <select id="tx-category" v-model="form.categoryId" class="field__input field__select">
              <option :value="undefined">Uncategorized</option>
              <option v-for="cat in categories" :key="cat.id" :value="cat.id">{{ cat.name }}</option>
            </select>
          </div>

          <!-- Notes -->
          <div class="field">
            <label class="field__label" for="tx-notes">Notes <span class="field__optional">(optional)</span></label>
            <textarea
              id="tx-notes"
              v-model="form.notes"
              class="field__input field__textarea"
              placeholder="Any extra detail…"
              rows="2"
            />
          </div>

          <!-- Error -->
          <p v-if="error" class="modal__error">{{ error }}</p>

          <!-- Footer -->
          <div class="modal__footer">
            <button type="button" class="btn btn--ghost" @click="$emit('close')">Cancel</button>
            <button type="submit" class="btn btn--primary" :disabled="busy">
              <Loader2 v-if="busy" :size="14" class="spin" />
              {{ isEdit ? 'Save changes' : 'Add transaction' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { X, Loader2 } from 'lucide-vue-next'
import type { Transaction } from '../../stores/transactions'
import type { Account }     from '../../stores/accounts'
import type { Category }    from '../../stores/categories'

const props = defineProps<{
  transaction?: Transaction
  accounts:     Account[]
  categories:   Category[]
  /** Pre-select account when creating */
  defaultAccountId?: number
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved', t: Partial<Transaction>): void
}>()

const isEdit = computed(() => !!props.transaction)
const busy   = ref(false)
const error  = ref<string | null>(null)

// ── Form state ────────────────────────────────────────────────────────────────
const txType       = ref<'expense' | 'income'>('expense')
const amountDisplay = ref('0.00')

interface FormState {
  date:       string
  payee:      string
  accountId:  number | ''
  categoryId: number | undefined
  notes:      string
}

function todayStr() {
  return new Date().toISOString().substring(0, 10)
}

function blankForm(): FormState {
  return {
    date:       todayStr(),
    payee:      '',
    accountId:  props.defaultAccountId ?? '',
    categoryId: undefined,
    notes:      '',
  }
}

const form = ref<FormState>(blankForm())

watch(
  () => props.transaction,
  (t) => {
    if (t) {
      const abs = Math.abs(t.amountCents)
      txType.value        = t.amountCents >= 0 ? 'income' : 'expense'
      amountDisplay.value = (abs / 100).toFixed(2)
      // date might be full ISO string or YYYY-MM-DD
      const dateStr = typeof t.date === 'string'
        ? t.date.substring(0, 10)
        : new Date(t.date).toISOString().substring(0, 10)
      form.value = {
        date:       dateStr,
        payee:      t.payee,
        accountId:  t.accountId,
        categoryId: t.categoryId ?? undefined,
        notes:      t.notes ?? '',
      }
    } else {
      form.value          = blankForm()
      txType.value        = 'expense'
      amountDisplay.value = '0.00'
    }
  },
  { immediate: true }
)

// ── Submit ────────────────────────────────────────────────────────────────────
async function submit() {
  error.value = null
  if (!form.value.accountId) { error.value = 'Please select an account.'; return }

  const raw    = parseFloat(amountDisplay.value || '0')
  const cents  = Math.round((isNaN(raw) ? 0 : raw) * 100)
  const signed = txType.value === 'expense' ? -cents : cents

  const payload: Partial<Transaction> = {
    ...(isEdit.value && props.transaction ? { id: props.transaction.id } : {}),
    date:        form.value.date as any,   // Go expects string; backend parses it
    payee:       form.value.payee.trim(),
    payeeNormalized: '',
    accountId:   form.value.accountId as number,
    amountCents: signed,
    categoryId:  form.value.categoryId,
    notes:       form.value.notes,
    isTransfer:  false,
    importHash:  props.transaction?.importHash ?? '',
    importSource: props.transaction?.importSource ?? 'manual',
  }

  busy.value = true
  try {
    emit('saved', payload)
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.modal-backdrop {
  position: fixed; inset: 0; z-index: 200;
  background: rgba(0,0,0,0.5);
  backdrop-filter: blur(4px);
  display: flex; align-items: center; justify-content: center;
  padding: var(--space-4);
  animation: fade-in var(--duration-base) var(--ease-out);
}
@keyframes fade-in { from { opacity: 0 } to { opacity: 1 } }

.modal {
  background: var(--color-surface-1);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  width: 100%; max-width: 440px;
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

.modal__title { font-size: 17px; font-weight: 600; color: var(--color-text-primary); }

.btn--icon {
  width: 32px; height: 32px; padding: 0;
  display: flex; align-items: center; justify-content: center;
}

.modal__body {
  padding: var(--space-5);
  display: flex; flex-direction: column; gap: var(--space-4);
}

/* Fields */
.field { display: flex; flex-direction: column; gap: var(--space-2); }
.field__label {
  font-size: 12px; font-weight: 600; letter-spacing: 0.04em;
  color: var(--color-text-secondary); text-transform: uppercase;
}
.field__optional { font-weight: 400; text-transform: none; letter-spacing: 0; }

.field__input {
  height: 38px; padding: 0 var(--space-3);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text-primary);
  font: var(--text-body);
  outline: none;
  transition: border-color var(--duration-fast), box-shadow var(--duration-fast);
  width: 100%; box-sizing: border-box;
}
.field__input:focus {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(26,138,97,0.2);
}

.field__textarea {
  height: auto; padding: var(--space-2) var(--space-3);
  resize: vertical; min-height: 60px;
}

.field__select {
  cursor: pointer; appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%235A6B60' stroke-width='2.5' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E");
  background-repeat: no-repeat; background-position: right var(--space-3) center;
  padding-right: 36px;
}

.field__input-wrap { position: relative; }
.field__prefix {
  position: absolute; left: var(--space-3); top: 50%; transform: translateY(-50%);
  color: var(--color-text-tertiary); font: var(--text-body); pointer-events: none;
}
.field__input--prefixed { padding-left: var(--space-6); }

/* Amount row */
.amount-row { display: flex; gap: var(--space-3); align-items: stretch; }

/* Type toggle */
.type-toggle {
  display: flex; border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  overflow: hidden; flex-shrink: 0;
}
.type-toggle__btn {
  padding: 0 var(--space-3); height: 38px;
  font: var(--text-body-sm); font-weight: 500;
  cursor: pointer; border: none;
  background: var(--color-surface-2);
  color: var(--color-text-secondary);
  transition: background var(--duration-fast), color var(--duration-fast);
}
.type-toggle__btn--active {
  background: var(--color-primary);
  color: var(--color-text-on-primary);
}

/* Error */
.modal__error {
  background: rgba(220,38,38,0.08);
  border: 1px solid rgba(220,38,38,0.25);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  font: var(--text-body-sm);
  color: var(--color-expense);
}

/* Footer */
.modal__footer {
  display: flex; justify-content: flex-end; gap: var(--space-3);
  padding-top: var(--space-2);
  border-top: 1px solid var(--color-border);
}

.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
