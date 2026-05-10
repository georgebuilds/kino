import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  ListAccounts,
  CreateAccount,
  UpdateAccount,
  DeleteAccount,
} from '../../wailsjs/go/main/App'
import type { models } from '../../wailsjs/go/models'

export type Account = models.Account

export const useAccountsStore = defineStore('accounts', () => {
  const accounts   = ref<Account[]>([])
  const loading    = ref(false)
  const error      = ref<string | null>(null)
  const initialised = ref(false)

  // ── Derived ──────────────────────────────────────────────────────────────

  const totalNetWorth = computed(() =>
    accounts.value.reduce((sum, a) => sum + a.balanceCents, 0)
  )

  const byType = computed(() => {
    const map: Record<string, Account[]> = {}
    for (const a of accounts.value) {
      ;(map[a.type] ??= []).push(a)
    }
    return map
  })

  // ── Actions ───────────────────────────────────────────────────────────────

  async function fetch() {
    if (loading.value) return
    loading.value = true
    error.value   = null
    try {
      accounts.value = await ListAccounts()
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to load accounts'
    } finally {
      loading.value  = false
      initialised.value = true
    }
  }

  async function create(draft: Omit<Account, 'id' | 'createdAt' | 'updatedAt'>) {
    const created = await CreateAccount(draft as Account)
    accounts.value.push(created)
    return created
  }

  async function update(acc: Account) {
    await UpdateAccount(acc)
    const idx = accounts.value.findIndex(a => a.id === acc.id)
    if (idx !== -1) accounts.value[idx] = acc
  }

  async function remove(id: number) {
    await DeleteAccount(id)
    accounts.value = accounts.value.filter(a => a.id !== id)
  }

  function findById(id: number) {
    return accounts.value.find(a => a.id === id) ?? null
  }

  return {
    accounts, loading, error, initialised,
    totalNetWorth, byType,
    fetch, create, update, remove, findById,
  }
})
