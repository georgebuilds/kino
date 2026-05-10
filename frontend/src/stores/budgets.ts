import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  GetBudgetPage,
  CreateBudget,
  UpdateBudget,
  DeleteBudget,
} from '../../wailsjs/go/main/App'
import type { main, models } from '../../wailsjs/go/models'

export type BudgetPage      = main.BudgetPage
export type BudgetLine      = main.BudgetLine
export type UnbudgetedLine  = main.UnbudgetedLine
export type Budget          = models.Budget

export const useBudgetsStore = defineStore('budgets', () => {
  const page    = ref<BudgetPage | null>(null)
  const loading = ref(false)
  const error   = ref<string | null>(null)

  // Active month
  const now = new Date()
  const year  = ref(now.getFullYear())
  const month = ref(now.getMonth() + 1)

  // ── Actions ───────────────────────────────────────────────────────────────

  async function fetch() {
    if (loading.value) return
    loading.value = true
    error.value   = null
    try {
      page.value = await GetBudgetPage(year.value, month.value)
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to load budgets'
    } finally {
      loading.value = false
    }
  }

  function prevMonth() {
    if (month.value === 1) { month.value = 12; year.value-- }
    else month.value--
    fetch()
  }

  function nextMonth() {
    if (month.value === 12) { month.value = 1; year.value++ }
    else month.value++
    fetch()
  }

  async function create(b: Omit<Budget, 'id'>) {
    const created = await CreateBudget(b as Budget)
    await fetch() // refetch so progress is correct
    return created
  }

  async function update(b: Budget) {
    await UpdateBudget(b)
    await fetch()
  }

  async function remove(id: number) {
    await DeleteBudget(id)
    if (page.value) {
      page.value = {
        ...page.value,
        lines: page.value.lines?.filter(l => l.id !== id) ?? [],
      } as BudgetPage
    }
  }

  return {
    page, loading, error, year, month,
    fetch, prevMonth, nextMonth,
    create, update, remove,
  }
})
