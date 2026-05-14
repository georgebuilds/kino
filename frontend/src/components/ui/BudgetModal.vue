<template>
  <Teleport to="body">
    <div class="modal-backdrop" @click.self="$emit('close')">
      <div ref="modalRef" class="modal" role="dialog" aria-modal="true" :aria-label="isEdit ? 'Edit budget' : 'Add budget'">

        <div class="modal__header">
          <h2 class="modal__title">{{ isEdit ? 'Edit budget' : 'Add budget' }}</h2>
          <button class="btn btn--ghost btn--icon" @click="$emit('close')" aria-label="Close">
            <X :size="16" />
          </button>
        </div>

        <form class="modal__body" @submit.prevent="submit">

          <!-- Category -->
          <div class="field">
            <label class="field__label" for="bud-cat">Category</label>
            <select
              id="bud-cat"
              v-model="form.categoryId"
              class="field__input field__select"
              required
              :disabled="isEdit"
            >
              <option value="" disabled>Select category…</option>
              <option
                v-for="cat in availableCategories"
                :key="cat.id"
                :value="cat.id"
              >{{ cat.name }}</option>
            </select>
            <p v-if="isEdit" class="field__hint">Category cannot be changed — delete and recreate to switch.</p>
          </div>

          <!-- Monthly amount -->
          <div class="field">
            <label class="field__label" for="bud-amount">Monthly budget</label>
            <div class="field__input-wrap">
              <span class="field__prefix">$</span>
              <input
                id="bud-amount"
                v-model="amountDisplay"
                class="field__input field__input--prefixed"
                type="number"
                step="0.01"
                min="0.01"
                placeholder="0.00"
                required
              />
            </div>
          </div>

          <!-- Rolls over toggle -->
          <div class="field field--row">
            <div class="field__toggle-text">
              <span class="field__label" style="margin-bottom:0">Roll over unspent</span>
              <span class="field__hint" style="margin-top:0">Add remaining budget to next month.</span>
            </div>
            <button
              type="button"
              class="toggle"
              :class="{ 'toggle--on': form.rollsOver }"
              @click="form.rollsOver = !form.rollsOver"
              role="switch"
              :aria-checked="form.rollsOver"
              aria-label="Roll over unspent budget"
            >
              <span class="toggle__knob" />
            </button>
          </div>

          <!-- Error -->
          <p v-if="error" class="modal__error">{{ error }}</p>

          <div class="modal__footer">
            <button type="button" class="btn btn--ghost" @click="$emit('close')">Cancel</button>
            <button type="submit" class="btn btn--primary" :disabled="busy">
              <Loader2 v-if="busy" :size="14" class="spin" />
              {{ isEdit ? 'Save changes' : 'Add budget' }}
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
import type { BudgetLine } from '../../stores/budgets'
import type { Category }   from '../../stores/categories'
import { useFocusTrap } from './useFocusTrap'

const props = defineProps<{
  line?:       BudgetLine      // defined = edit, undefined = create
  categories:  Category[]
  /** IDs already budgeted — excluded from the "new budget" dropdown */
  budgetedIds: number[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved', payload: { categoryId: number; amountCents: number; rollsOver: boolean }): void
}>()

const isEdit   = computed(() => !!props.line)
const busy     = ref(false)
const error    = ref<string | null>(null)
const modalRef = ref<HTMLElement | null>(null)

useFocusTrap(modalRef)

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}
onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => window.removeEventListener('keydown', onKeydown))

// Categories available to budget (expense only, not already budgeted — unless editing)
const availableCategories = computed(() => {
  return props.categories.filter(c => {
    if (c.isIncome || c.name === 'Transfers') return false
    if (isEdit.value) return true          // editing: show current cat
    return !props.budgetedIds.includes(c.id)
  })
})

// Form
interface FormState {
  categoryId: number | ''
  rollsOver:  boolean
}

const form = ref<FormState>({ categoryId: '', rollsOver: false })
const amountDisplay = ref('0.00')

watch(
  () => props.line,
  (line) => {
    if (line) {
      form.value   = { categoryId: line.categoryId, rollsOver: line.rollsOver }
      amountDisplay.value = (line.budgetCents / 100).toFixed(2)
    } else {
      form.value   = { categoryId: '', rollsOver: false }
      amountDisplay.value = '0.00'
    }
  },
  { immediate: true }
)

async function submit() {
  error.value = null
  if (!form.value.categoryId) { error.value = 'Please select a category.'; return }
  const raw = parseFloat(amountDisplay.value || '0')
  if (isNaN(raw) || raw <= 0) { error.value = 'Please enter a valid amount.'; return }

  busy.value = true
  emit('saved', {
    categoryId:  form.value.categoryId as number,
    amountCents: Math.round(raw * 100),
    rollsOver:   form.value.rollsOver,
  })
  // busy stays true until the parent closes this modal
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

.modal {
  background: var(--color-surface-1);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  width: 100%; max-width: 400px;
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
.field--row {
  flex-direction: row; align-items: center;
  justify-content: space-between; gap: var(--space-3);
}

.field__label {
  font-size: 12px; font-weight: 600; letter-spacing: 0.04em;
  color: var(--color-text-secondary); text-transform: uppercase;
  display: block;
}
.field__hint {
  font: var(--text-body-sm); color: var(--color-text-tertiary); margin: 0;
}

.field__toggle-text { display: flex; flex-direction: column; gap: 2px; }

.field__input {
  height: 38px; padding: 0 var(--space-3);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text-primary);
  font: var(--text-body); outline: none;
  transition: border-color var(--duration-fast), box-shadow var(--duration-fast);
  width: 100%; box-sizing: border-box;
}
.field__input:focus {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(26,138,97,0.2);
}
.field__input:disabled { opacity: 0.5; cursor: not-allowed; }

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

/* Toggle switch */
.toggle {
  width: 40px; height: 22px; border-radius: 11px;
  background: var(--color-surface-2);
  border: 1.5px solid var(--color-border);
  cursor: pointer; position: relative;
  flex-shrink: 0;
  transition: background var(--duration-base), border-color var(--duration-base);
}
.toggle--on { background: var(--color-primary); border-color: var(--color-primary); }

.toggle__knob {
  display: block; width: 16px; height: 16px; border-radius: 50%;
  background: white;
  position: absolute; top: 2px; left: 2px;
  transition: transform var(--duration-base) var(--ease-out);
  box-shadow: 0 1px 3px rgba(0,0,0,0.2);
}
.toggle--on .toggle__knob { transform: translateX(18px); }

/* Error */
.modal__error {
  background: rgba(220,38,38,0.08);
  border: 1px solid rgba(220,38,38,0.25);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  font: var(--text-body-sm); color: var(--color-expense);
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
