/**
 * Shared map: kebab-case icon name string ↔ lucide-vue-next component.
 * Keep this as the single source of truth for anywhere that renders category icons.
 */
import type { Component } from 'vue'
import {
  Tag, Home, Car, Utensils, HeartPulse, ShoppingBag, Tv, Ticket,
  PiggyBank, TrendingUp, ArrowLeftRight, Banknote, Coffee, Book,
  GraduationCap, Plane, Wrench, Smartphone, Baby, Dumbbell,
  Music, Briefcase, Gift, CreditCard, Gamepad2, Film, Bike,
  Wallet, Receipt, Coins, Fuel, Bus, Heart, Star, Globe,
  Zap, Umbrella, Dog, Shirt, Hotel, Stethoscope,
} from 'lucide-vue-next'

export interface IconOption {
  name: string
  component: Component
}

export const ICON_OPTIONS: IconOption[] = [
  // Finance
  { name: 'tag',              component: Tag },
  { name: 'banknote',         component: Banknote },
  { name: 'credit-card',      component: CreditCard },
  { name: 'receipt',          component: Receipt },
  { name: 'wallet',           component: Wallet },
  { name: 'coins',            component: Coins },
  { name: 'piggy-bank',       component: PiggyBank },
  { name: 'trending-up',      component: TrendingUp },
  // Home / utilities
  { name: 'home',             component: Home },
  { name: 'hotel',            component: Hotel },
  { name: 'wrench',           component: Wrench },
  { name: 'zap',              component: Zap },
  { name: 'umbrella',         component: Umbrella },
  // Transport
  { name: 'car',              component: Car },
  { name: 'plane',            component: Plane },
  { name: 'bus',              component: Bus },
  { name: 'bike',             component: Bike },
  { name: 'fuel',             component: Fuel },
  // Food & drink
  { name: 'utensils',         component: Utensils },
  { name: 'coffee',           component: Coffee },
  // Shopping & lifestyle
  { name: 'shopping-bag',     component: ShoppingBag },
  { name: 'shirt',            component: Shirt },
  { name: 'gift',             component: Gift },
  // Health
  { name: 'heart-pulse',      component: HeartPulse },
  { name: 'heart',            component: Heart },
  { name: 'stethoscope',      component: Stethoscope },
  { name: 'dumbbell',         component: Dumbbell },
  { name: 'baby',             component: Baby },
  // Entertainment & education
  { name: 'tv',               component: Tv },
  { name: 'ticket',           component: Ticket },
  { name: 'music',            component: Music },
  { name: 'film',             component: Film },
  { name: 'gamepad-2',        component: Gamepad2 },
  { name: 'book',             component: Book },
  { name: 'graduation-cap',   component: GraduationCap },
  // Work & tech
  { name: 'briefcase',        component: Briefcase },
  { name: 'smartphone',       component: Smartphone },
  // Other
  { name: 'arrow-left-right', component: ArrowLeftRight },
  { name: 'dog',              component: Dog },
  { name: 'star',             component: Star },
  { name: 'globe',            component: Globe },
]

export const ICON_MAP: Record<string, Component> = Object.fromEntries(
  ICON_OPTIONS.map(o => [o.name, o.component])
)

/** Returns the lucide component for a stored icon name, falling back to Tag. */
export function iconComponent(name: string): Component {
  return ICON_MAP[name] ?? Tag
}

// ── Color palette ─────────────────────────────────────────────────────────────

/** 18 hand-picked swatches that look good in the Kino dark theme. */
export const COLOR_SWATCHES: string[] = [
  // Row 1 — greens / teals / blues
  '#1A8A61', '#2A9D8F', '#4A9E8A', '#2A7FA8', '#1D6FA0', '#3B82F6',
  // Row 2 — purples / pinks / reds
  '#6D4C9E', '#8B5CF6', '#B84A72', '#EC4899', '#EF4444', '#C4603A',
  // Row 3 — ambers / neutrals / dark
  '#F97316', '#C4943A', '#A87A28', '#EAB308', '#5A6B60', '#374151',
]
