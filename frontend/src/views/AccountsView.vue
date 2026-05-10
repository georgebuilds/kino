<template>
  <div class="page">

    <!-- Header -->
    <div class="page-header flex items-center justify-between">
      <div>
        <h1 class="page-title">Accounts</h1>
        <p class="page-subtitle">{{ formatMoney(accountsStore.totalNetWorth) }} net worth</p>
      </div>
      <button class="btn btn--primary btn--sm" @click="openCreate">
        <Plus :size="14" />
        Add account
      </button>
    </div>

    <!-- Loading skeleton -->
    <div v-if="accountsStore.loading" class="accounts-skeleton">
      <div v-for="i in 4" :key="i" class="skeleton-row" />
    </div>

    <!-- Empty state -->
    <div v-else-if="accountsStore.accounts.length === 0" class="accounts-empty card">
      <div class="accounts-empty__icon">
        <Landmark :size="28" />
      </div>
      <h2 class="accounts-empty__title">No accounts yet</h2>
      <p class="accounts-empty__sub">Add your checking, savings, investment and other accounts to track your net worth.</p>
      <button class="btn btn--primary" @click="openCreate">
        <Plus :size="14" />
        Add your first account
      </button>
    </div>

    <!-- Account groups -->
    <div v-else class="accounts-groups">

      <!-- Assets -->
      <section v-if="assets.length" class="accounts-section">
        <div class="accounts-section__header">
          <span class="accounts-section__label">Assets</span>
          <span class="accounts-section__total tabular-nums">{{ formatMoney(assetsTotal) }}</span>
        </div>
        <div class="card accounts-card">
          <div
            v-for="(acc, idx) in assets"
            :key="acc.id"
            class="accounts-card__row"
            :class="{ 'accounts-card__row--last': idx === assets.length - 1 }"
          >
            <AccountRow :account="acc" />
            <div class="accounts-card__actions">
              <button class="btn btn--ghost btn--icon" @click="openEdit(acc)" aria-label="Edit">
                <Pencil :size="13" />
              </button>
              <button class="btn btn--ghost btn--icon btn--danger" @click="confirmDelete(acc)" aria-label="Delete">
                <Trash2 :size="13" />
              </button>
            </div>
          </div>
        </div>
      </section>

      <!-- Liabilities -->
      <section v-if="liabilities.length" class="accounts-section">
        <div class="accounts-section__header">
          <span class="accounts-section__label">Liabilities</span>
          <span class="accounts-section__total accounts-section__total--neg tabular-nums">{{ formatMoney(liabilitiesTotal) }}</span>
        </div>
        <div class="card accounts-card">
          <div
            v-for="(acc, idx) in liabilities"
            :key="acc.id"
            class="accounts-card__row"
            :class="{ 'accounts-card__row--last': idx === liabilities.length - 1 }"
          >
            <AccountRow :account="acc" />
            <div class="accounts-card__actions">
              <button class="btn btn--ghost btn--icon" @click="openEdit(acc)" aria-label="Edit">
                <Pencil :size="13" />
              </button>
              <button class="btn btn--ghost btn--icon btn--danger" @click="confirmDelete(acc)" aria-label="Delete">
                <Trash2 :size="13" />
              </button>
            </div>
          </div>
        </div>
      </section>

    </div>

    <!-- Delete confirmation -->
    <Teleport to="body">
      <div v-if="deleteTarget" class="modal-backdrop" @click.self="deleteTarget = null">
        <div class="modal modal--sm" role="alertdialog">
          <div class="modal__header">
            <h2 class="modal__title">Delete account?</h2>
          </div>
          <div class="modal__body">
            <p class="text-body" style="color: var(--color-text-secondary);">
              <strong style="color:var(--color-text-primary)">{{ deleteTarget.name }}</strong> and all its transactions will be permanently removed. This cannot be undone.
            </p>
            <p v-if="deleteError" class="modal__error">{{ deleteError }}</p>
            <div class="modal__footer">
              <button class="btn btn--ghost" @click="deleteTarget = null">Cancel</button>
              <button class="btn btn--danger" :disabled="deleteBusy" @click="doDelete">
                <Loader2 v-if="deleteBusy" :size="14" class="spin" />
                Delete account
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Add / Edit modal -->
    <AccountModal
      v-if="showModal"
      :account="editTarget ?? undefined"
      @close="closeModal"
      @saved="onSaved"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { Plus, Pencil, Trash2, Loader2, Landmark } from 'lucide-vue-next'
import AccountRow from '../components/ui/AccountRow.vue'
import AccountModal from '../components/ui/AccountModal.vue'
import { useAccountsStore } from '../stores/accounts'
import type { Account } from '../stores/accounts'

const accountsStore = useAccountsStore()

// ── Grouping ──────────────────────────────────────────────────────────────────
const ASSET_TYPES     = new Set(['checking', 'savings', 'investment', 'crypto', 'cash', 'other'])
const LIABILITY_TYPES = new Set(['credit_card', 'loan'])

const assets      = computed(() => accountsStore.accounts.filter(a => ASSET_TYPES.has(a.type)))
const liabilities = computed(() => accountsStore.accounts.filter(a => LIABILITY_TYPES.has(a.type)))

const assetsTotal      = computed(() => assets.value.reduce((s, a) => s + a.balanceCents, 0))
const liabilitiesTotal = computed(() => liabilities.value.reduce((s, a) => s + a.balanceCents, 0))

// ── Formatting ─────────────────────────────────────────────────────────────────
function formatMoney(cents: number) {
  const abs = Math.abs(cents)
  const formatted = (abs / 100).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
  return (cents < 0 ? '-$' : '$') + formatted
}

// ── Create / Edit modal ───────────────────────────────────────────────────────
const showModal  = ref(false)
const editTarget = ref<Account | null>(null)

function openCreate() {
  editTarget.value = null
  showModal.value  = true
}

function openEdit(acc: Account) {
  editTarget.value = { ...acc } as Account
  showModal.value  = true
}

function closeModal() {
  showModal.value  = false
  editTarget.value = null
}

async function onSaved(draft: Account) {
  if (editTarget.value) {
    // Merge draft onto original; preserve id, timestamps, balanceCents (not editable in modal)
    const merged = { ...editTarget.value, ...draft, id: editTarget.value.id } as Account
    await accountsStore.update(merged)
  } else {
    await accountsStore.create(draft)
  }
  closeModal()
}

// ── Delete ────────────────────────────────────────────────────────────────────
const deleteTarget = ref<Account | null>(null)
const deleteBusy   = ref(false)
const deleteError  = ref<string | null>(null)

function confirmDelete(acc: Account) {
  deleteError.value  = null
  deleteTarget.value = acc
}

async function doDelete() {
  if (!deleteTarget.value) return
  deleteBusy.value  = true
  deleteError.value = null
  try {
    await accountsStore.remove(deleteTarget.value.id)
    deleteTarget.value = null
  } catch (e: any) {
    deleteError.value = e?.message ?? 'Failed to delete account'
  } finally {
    deleteBusy.value = false
  }
}
</script>

<style scoped>
/* ── Section layout ── */
.accounts-groups {
  display: flex; flex-direction: column; gap: var(--space-6);
  padding-bottom: var(--space-8);
}

.accounts-section__header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 0 var(--space-1);
  margin-bottom: var(--space-3);
}

.accounts-section__label {
  font-size: 11px; font-weight: 700; letter-spacing: 0.08em;
  text-transform: uppercase; color: var(--color-text-tertiary);
}

.accounts-section__total {
  font: var(--text-body); font-weight: 600;
  color: var(--color-text-primary);
}

.accounts-section__total--neg { color: var(--color-expense); }

/* ── Card rows ── */
.accounts-card { padding: 0; overflow: hidden; }

.accounts-card__row {
  display: flex; align-items: center;
  padding: 0 var(--space-3) 0 var(--space-2);
  border-bottom: 1px solid var(--color-border);
  position: relative;
}

.accounts-card__row--last { border-bottom: none; }

.accounts-card__row :deep(.account-row) {
  flex: 1;
  border-radius: 0;
}

/* Actions reveal on hover */
.accounts-card__actions {
  display: flex; gap: var(--space-1);
  opacity: 0;
  transition: opacity var(--duration-fast) var(--ease-out);
  flex-shrink: 0;
}

.accounts-card__row:hover .accounts-card__actions { opacity: 1; }

.btn--icon {
  width: 30px; height: 30px; padding: 0;
  display: flex; align-items: center; justify-content: center;
}

.btn--danger { color: var(--color-expense); }
.btn--danger:hover { background: rgba(220,38,38,0.08); }

/* ── Empty state ── */
.accounts-empty {
  display: flex; flex-direction: column; align-items: center;
  text-align: center; gap: var(--space-4);
  padding: var(--space-12) var(--space-8);
  margin-top: var(--space-4);
}

.accounts-empty__icon {
  width: 56px; height: 56px; border-radius: var(--radius-lg);
  background: var(--color-surface-2);
  display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary);
}

.accounts-empty__title {
  font: var(--text-heading); color: var(--color-text-primary);
}

.accounts-empty__sub {
  font: var(--text-body); color: var(--color-text-secondary);
  max-width: 340px; line-height: 1.6;
}

/* ── Skeleton loader ── */
.accounts-skeleton { display: flex; flex-direction: column; gap: var(--space-2); margin-top: var(--space-4); }
.skeleton-row {
  height: 60px; border-radius: var(--radius-md);
  background: linear-gradient(90deg, var(--color-surface-2) 25%, var(--color-surface-1) 50%, var(--color-surface-2) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite;
}
@keyframes shimmer { to { background-position: -200% 0 } }

/* ── Shared modal pieces (delete confirm) ── */
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
  display: flex; align-items: center; justify-content: space-between;
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
  font: var(--text-body-sm);
  color: var(--color-expense);
}

.modal__footer {
  display: flex; justify-content: flex-end; gap: var(--space-3);
  padding-top: var(--space-2);
  border-top: 1px solid var(--color-border);
}

.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
