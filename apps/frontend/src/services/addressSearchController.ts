import {
  geocodeLocation,
  PRESET_PLACES,
  type PlaceSuggestion,
} from './spotsApi.ts'

export interface AddressSearchControllerOptions {
  initialPlace?: PlaceSuggestion
  onSelect?: (place: PlaceSuggestion) => Promise<void> | void
  onStateChange?: () => void
}

export class AddressSearchController {
  searchQuery: string
  suggestions: PlaceSuggestion[]
  showSuggestions: boolean
  searchingAddress: boolean
  activeSuggestionIndex: number
  selectedPlace: PlaceSuggestion

  private debounceTimer: ReturnType<typeof setTimeout> | null = null
  private currentAbortController: AbortController | null = null
  private onSelectCallback?: (place: PlaceSuggestion) => Promise<void> | void
  private onStateChangeCallback?: () => void

  constructor(options: AddressSearchControllerOptions = {}) {
    const initial = options.initialPlace || PRESET_PLACES[0]
    this.searchQuery = initial.name
    this.suggestions = PRESET_PLACES
    this.showSuggestions = false
    this.searchingAddress = false
    this.activeSuggestionIndex = -1
    this.selectedPlace = initial
    this.onSelectCallback = options.onSelect
    this.onStateChangeCallback = options.onStateChange
  }

  private notify() {
    if (this.onStateChangeCallback) {
      this.onStateChangeCallback()
    }
  }

  handleSearchInput(newQuery?: string) {
    if (newQuery !== undefined) {
      this.searchQuery = newQuery
    }
    this.showSuggestions = true
    this.activeSuggestionIndex = -1

    if (this.debounceTimer) {
      clearTimeout(this.debounceTimer)
    }
    if (this.currentAbortController) {
      this.currentAbortController.abort()
    }

    const q = this.searchQuery.trim()
    if (!q) {
      this.suggestions = PRESET_PLACES
      this.searchingAddress = false
      this.notify()
      return
    }

    this.searchingAddress = true
    this.notify()

    this.debounceTimer = setTimeout(async () => {
      this.currentAbortController = new AbortController()
      try {
        const results = await geocodeLocation(q, this.currentAbortController.signal)
        this.suggestions = results
      } catch (err: any) {
        if (err?.name !== 'AbortError') {
          this.suggestions = PRESET_PLACES
        }
      } finally {
        this.searchingAddress = false
        this.notify()
      }
    }, 280)
  }

  clearSearch() {
    this.searchQuery = ''
    this.suggestions = PRESET_PLACES
    this.activeSuggestionIndex = -1
    this.showSuggestions = true
    this.notify()
  }

  async selectDestination(place: PlaceSuggestion) {
    this.selectedPlace = place
    this.searchQuery = place.name
    this.showSuggestions = false
    this.activeSuggestionIndex = -1
    this.notify()

    if (this.onSelectCallback) {
      await this.onSelectCallback(place)
    }
  }

  async submitSearch() {
    const q = this.searchQuery.trim()
    if (!q) {
      await this.selectDestination(this.selectedPlace)
      return
    }

    if (
      this.showSuggestions &&
      this.activeSuggestionIndex >= 0 &&
      this.suggestions[this.activeSuggestionIndex]
    ) {
      await this.selectDestination(this.suggestions[this.activeSuggestionIndex])
      return
    }

    if (q.toLowerCase() !== this.selectedPlace.name.toLowerCase()) {
      this.searchingAddress = true
      this.notify()
      try {
        const results = await geocodeLocation(q)
        this.suggestions = results
        if (results.length > 0) {
          await this.selectDestination(results[0])
          return
        }
      } finally {
        this.searchingAddress = false
        this.notify()
      }
    }

    await this.selectDestination(this.selectedPlace)
  }

  handleKeydown(e: { key: string; preventDefault?: () => void }) {
    if (!this.showSuggestions || this.suggestions.length === 0) return

    if (e.key === 'ArrowDown') {
      if (e.preventDefault) e.preventDefault()
      this.activeSuggestionIndex =
        (this.activeSuggestionIndex + 1) % this.suggestions.length
      this.notify()
    } else if (e.key === 'ArrowUp') {
      if (e.preventDefault) e.preventDefault()
      this.activeSuggestionIndex =
        this.activeSuggestionIndex <= 0
          ? this.suggestions.length - 1
          : this.activeSuggestionIndex - 1
      this.notify()
    } else if (e.key === 'Enter') {
      if (e.preventDefault) e.preventDefault()
      void this.submitSearch()
    } else if (e.key === 'Escape') {
      this.showSuggestions = false
      this.notify()
    }
  }

  dispose() {
    if (this.debounceTimer) {
      clearTimeout(this.debounceTimer)
      this.debounceTimer = null
    }
    if (this.currentAbortController) {
      this.currentAbortController.abort()
      this.currentAbortController = null
    }
  }
}
