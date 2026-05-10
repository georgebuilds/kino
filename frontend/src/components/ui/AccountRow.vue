<template>
  <div class="account-row">
    <div class="account-row__icon" :class="`account-row__icon--${account.type}`">
      <component :is="icon" :size="15" />
    </div>
    <div class="account-row__info">
      <p class="text-body account-row__name">{{ account.name }}</p>
      <p class="text-body-sm account-row__type">{{ account.institution || typeLabel }}</p>
    </div>
    <span class="account-row__balance tabular-nums" :class="account.balanceCents < 0 ? 'amount--negative' : ''">
      {{ account.balanceCents < 0 ? '-' : '' }}${{ formatBalance(account.balanceCents) }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed, markRaw } from 'vue'
import { Building2, PiggyBank, TrendingUp, Bitcoin, CreditCard, Banknote, HelpCircle } from 'lucide-vue-next'
import type { models } from '../../../wailsjs/go/models'

type Account = models.Account

const props = defineProps<{ account: Account }>()

const iconMap: Record<string, any> = {
  checking:    markRaw(Building2),
  savings:     markRaw(PiggyBank),
  investment:  markRaw(TrendingUp),
  crypto:      markRaw(Bitcoin),
  credit_card: markRaw(CreditCard),
  loan:        markRaw(Banknote),
  cash:        markRaw(Banknote),
  other:       markRaw(HelpCircle),
}

const typeLabels: Record<string, string> = {
  checking: 'Checking', savings: 'Savings', investment: 'Investment',
  crypto: 'Crypto', credit_card: 'Credit Card', loan: 'Loan',
  cash: 'Cash', other: 'Other',
}

const icon      = computed(() => iconMap[props.account.type]  ?? markRaw(HelpCircle))
const typeLabel = computed(() => typeLabels[props.account.type] ?? props.account.type)

function formatBalance(cents: number) {
  return (Math.abs(cents) / 100).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
</script>

<style scoped>
.account-row {
  display: flex; align-items: center; gap: var(--space-3);
  padding: var(--space-3) var(--space-2);
  border-radius: var(--radius-md);
  transition: background-color var(--duration-fast) var(--ease-out);
  cursor: default;
}
.account-row:hover { background: var(--color-surface-2); }

.account-row__icon {
  width: 32px; height: 32px; border-radius: var(--radius-md);
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.account-row__icon--checking    { background: rgba(42,127,168,0.12); color: #2A7FA8; }
.account-row__icon--savings     { background: rgba(26,138,97,0.12);  color: var(--color-primary); }
.account-row__icon--investment  { background: rgba(109,76,158,0.12); color: #6D4C9E; }
.account-row__icon--crypto      { background: rgba(196,148,58,0.12); color: var(--color-accent); }
.account-row__icon--credit_card { background: rgba(184,74,114,0.12); color: #B84A72; }
.account-row__icon--loan        { background: rgba(220,38,38,0.10);  color: var(--color-expense); }
.account-row__icon--cash        { background: rgba(26,138,97,0.08);  color: var(--color-primary); }
.account-row__icon--other       { background: var(--color-surface-2); color: var(--color-text-tertiary); }

.account-row__info { flex: 1; min-width: 0; }
.account-row__name {
  font-weight: 500; color: var(--color-text-primary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.account-row__type { color: var(--color-text-tertiary); margin-top: 1px; }
.account-row__balance {
  font: var(--text-body); font-weight: 600;
  color: var(--color-text-primary);
  white-space: nowrap; font-variant-numeric: tabular-nums;
}
</style>
