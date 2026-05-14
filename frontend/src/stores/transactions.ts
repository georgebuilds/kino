import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  ListTransactions,
  CreateTransaction,
  UpdateTransaction,
  DeleteTransaction,
} from '../../wailsjs/go/main/App'
import type { models } from '../../wailsjs/go/models'
import type { db } from '../../wailsjs/go/models'

export type Transaction = models.Transaction
export type Filter      = db.TxFilter

const DEFAULT_LIMIT = 100

export const useTransactionsStore = defineStore('transactions', () => {
  const transactions = ref<Transaction[]>([])
  const total        = ref(0)
  const loading      = ref(false)
  const error        = ref<string | null>(null)

  // Active filter — mutate then call fetch()
  const filter = ref<Filter>({
    accountId:  undefined,
    categoryId: undefined,
    dateFrom:   '',
    dateTo:     '',
    search:     '',
    limit:      DEFAULT_LIMIT,
    offset:     0,
  })

  // ── Actions ───────────────────────────────────────────────────────────────

  let seq = 0

  async function fetch() {
    const my = ++seq
    loading.value = true
    error.value   = null
    try {
      const page = await ListTransactions(filter.value)
      if (my !== seq) return
      transactions.value = page.transactions
      total.value        = page.total
    } catch (e: any) {
      if (my !== seq) return
      error.value = e?.message ?? 'Failed to load transactions'
    } finally {
      if (my === seq) loading.value = false
    }
  }

  async function fetchNextPage() {
    filter.value.offset = transactions.value.length
    const res = await ListTransactions(filter.value)
    transactions.value.push(...res.transactions)
    total.value = res.total
  }

  function resetFilter() {
    filter.value = {
      accountId:  undefined,
      categoryId: undefined,
      dateFrom:   '',
      dateTo:     '',
      search:     '',
      limit:      DEFAULT_LIMIT,
      offset:     0,
    }
  }

  async function create(draft: Omit<Transaction, 'id' | 'createdAt' | 'updatedAt'>) {
    const created = await CreateTransaction(draft as Transaction)
    // Prepend so it appears at top of the list if in range.
    transactions.value.unshift(created)
    total.value++
    return created
  }

  async function update(t: Transaction) {
    await UpdateTransaction(t)
    const idx = transactions.value.findIndex(x => x.id === t.id)
    if (idx !== -1) transactions.value[idx] = t
  }

  async function remove(id: number) {
    await DeleteTransaction(id)
    transactions.value = transactions.value.filter(t => t.id !== id)
    total.value = Math.max(0, total.value - 1)
  }

  return {
    transactions, total, loading, error, filter,
    fetch, fetchNextPage, resetFilter,
    create, update, remove,
  }
})
