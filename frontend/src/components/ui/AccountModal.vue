<template>
  <Teleport to="body">
    <div class="modal-backdrop" @click.self="$emit('close')">
      <div ref="modalRef" class="modal" role="dialog" aria-modal="true" :aria-label="isEdit ? 'Edit account' : 'Add account'">
        <!-- Header -->
        <div class="modal__header">
          <h2 class="modal__title">{{ isEdit ? 'Edit account' : 'Add account' }}</h2>
          <button class="modal__close btn btn--ghost btn--icon" @click="$emit('close')" aria-label="Close">
            <X :size="16" />
          </button>
        </div>

        <!-- Form -->
        <form class="modal__body" @submit.prevent="submit">
          <!-- Name -->
          <div class="field">
            <label class="field__label" for="acc-name">Account name</label>
            <input
              id="acc-name"
              v-model="form.name"
              class="field__input"
              placeholder="e.g. Chase Checking"
              required
              autocomplete="off"
            />
          </div>

          <!-- Type -->
          <div class="field">
            <label class="field__label" for="acc-type">Account type</label>
            <select id="acc-type" v-model="form.type" class="field__input field__select">
              <option v-for="opt in typeOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </option>
            </select>
          </div>

          <!-- Institution -->
          <div class="field">
            <label class="field__label" for="acc-institution">Institution <span class="field__optional">(optional)</span></label>
            <input
              id="acc-institution"
              v-model="form.institution"
              class="field__input"
              placeholder="e.g. Chase, Fidelity"
              autocomplete="off"
            />
          </div>

          <!-- Opening balance — only shown for new accounts -->
          <div v-if="!isEdit" class="field">
            <label class="field__label" for="acc-balance">Opening balance</label>
            <div class="field__input-wrap">
              <span class="field__prefix">$</span>
              <input
                id="acc-balance"
                v-model="balanceDisplay"
                class="field__input field__input--prefixed"
                type="number"
                step="0.01"
                placeholder="0.00"
              />
            </div>
            <p class="field__hint">
              {{ isLiability ? 'Enter what you currently owe as a positive number.' : 'Enter your current balance.' }}
            </p>
          </div>

          <!-- Error banner -->
          <p v-if="error" class="modal__error">{{ error }}</p>

          <!-- Footer -->
          <div class="modal__footer">
            <button type="button" class="btn btn--ghost" @click="$emit('close')">Cancel</button>
            <button type="submit" class="btn btn--primary" :disabled="busy">
              <Loader2 v-if="busy" :size="14" class="spin" />
              {{ isEdit ? 'Save changes' : 'Add account' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { X, Loader2 } from 'lucide-vue-next'
import type { Account } from '../../stores/accounts'
import { useFocusTrap } from './useFocusTrap'

// ── Props / emits ─────────────────────────────────────────────────────────────
const props = defineProps<{
  account?: Account   // undefined = create mode, defined = edit mode
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved', account: Account): void
}>()

// ── Local state ───────────────────────────────────────────────────────────────
const isEdit   = computed(() => !!props.account)
const busy     = ref(false)
const error    = ref<string | null>(null)
const modalRef = ref<HTMLElement | null>(null)

useFocusTrap(modalRef)

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}
onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => window.removeEventListener('keydown', onKeydown))

const typeOptions = [
  { value: 'checking',    label: 'Checking' },
  { value: 'savings',     label: 'Savings' },
  { value: 'investment',  label: 'Investment' },
  { value: 'crypto',      label: 'Crypto' },
  { value: 'credit_card', label: 'Credit Card' },
  { value: 'loan',        label: 'Loan' },
  { value: 'cash',        label: 'Cash' },
  { value: 'other',       label: 'Other' },
]

const liabilityTypes = new Set(['credit_card', 'loan'])
const isLiability = computed(() => liabilityTypes.has(form.value.type))

// ── Form state ────────────────────────────────────────────────────────────────
function blankForm() {
  return {
    name:        '',
    type:        'checking' as Account['type'],
    institution: '',
    currency:    'USD',
    isHidden:    false,
    sortOrder:   0,
    balanceCents: 0,
    lastSyncedAt: null as null,
  }
}

const form = ref(blankForm())
const balanceDisplay = ref('0.00')

// Populate form when editing
watch(
  () => props.account,
  (acc) => {
    if (acc) {
      form.value = {
        name:        acc.name,
        type:        acc.type,
        institution: acc.institution ?? '',
        currency:    acc.currency ?? 'USD',
        isHidden:    acc.isHidden ?? false,
        sortOrder:   acc.sortOrder ?? 0,
        balanceCents: acc.balanceCents,
        lastSyncedAt: null,
      }
    } else {
      form.value = blankForm()
      balanceDisplay.value = '0.00'
    }
  },
  { immediate: true }
)

// ── Submit ────────────────────────────────────────────────────────────────────
async function submit() {
  error.value = null

  // Parse opening balance for new accounts
  if (!isEdit.value) {
    const raw = parseFloat(balanceDisplay.value || '0')
    const cents = Math.round((isNaN(raw) ? 0 : raw) * 100)
    // Liabilities: store as negative cents (owed)
    form.value.balanceCents = isLiability.value ? -Math.abs(cents) : cents
  }

  busy.value = true
  emit('saved', form.value as unknown as Account)
  // busy stays true until the parent closes this modal
}
</script>

<style scoped>
/* ── Backdrop ── */
.modal-backdrop {
  position: fixed; inset: 0; z-index: 200;
  background: rgba(0,0,0,0.5);
  backdrop-filter: blur(4px);
  display: flex; align-items: center; justify-content: center;
  padding: var(--space-4);
  animation: fade-in var(--duration-base) var(--ease-out);
}

@keyframes fade-in { from { opacity: 0 } to { opacity: 1 } }

/* ── Modal shell ── */
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

/* ── Header ── */
.modal__header {
  display: flex; align-items: center; justify-content: space-between;
  padding: var(--space-5) var(--space-5) 0;
}

.modal__title {
  font: var(--text-heading);
  color: var(--color-text-primary);
  font-size: 17px;
}

.btn--icon {
  width: 32px; height: 32px; padding: 0;
  display: flex; align-items: center; justify-content: center;
}

/* ── Body / form ── */
.modal__body {
  padding: var(--space-5);
  display: flex; flex-direction: column; gap: var(--space-4);
}

/* ── Fields ── */
.field { display: flex; flex-direction: column; gap: var(--space-2); }

.field__label {
  font-size: 12px; font-weight: 600; letter-spacing: 0.04em;
  color: var(--color-text-secondary); text-transform: uppercase;
}

.field__optional { font-weight: 400; text-transform: none; letter-spacing: 0; }

.field__input {
  height: 38px;
  padding: 0 var(--space-3);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text-primary);
  font: var(--text-body);
  outline: none;
  transition: border-color var(--duration-fast) var(--ease-out),
              box-shadow   var(--duration-fast) var(--ease-out);
  width: 100%;
  box-sizing: border-box;
}

.field__input:focus {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(26,138,97,0.2);
}

.field__select { cursor: pointer; appearance: none; background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%235A6B60' stroke-width='2.5' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E"); background-repeat: no-repeat; background-position: right var(--space-3) center; padding-right: 36px; }

.field__input-wrap { position: relative; }
.field__prefix {
  position: absolute; left: var(--space-3); top: 50%; transform: translateY(-50%);
  color: var(--color-text-tertiary); font: var(--text-body); pointer-events: none;
}
.field__input--prefixed { padding-left: var(--space-6); }

.field__hint {
  font: var(--text-body-sm);
  color: var(--color-text-tertiary);
  margin: 0;
}

/* ── Error ── */
.modal__error {
  background: rgba(220,38,38,0.08);
  border: 1px solid rgba(220,38,38,0.25);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  font: var(--text-body-sm);
  color: var(--color-expense);
}

/* ── Footer ── */
.modal__footer {
  display: flex; justify-content: flex-end; gap: var(--space-3);
  padding-top: var(--space-2);
  border-top: 1px solid var(--color-border);
  margin-top: var(--space-2);
}

/* ── Spinner ── */
.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
