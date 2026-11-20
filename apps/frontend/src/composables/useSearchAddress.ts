import { ref, onBeforeUnmount } from 'vue'
import {
  AddressSearchController,
  type AddressSearchControllerOptions,
} from '../services/addressSearchController'
import type { PlaceSuggestion } from '../services/spotsApi'

export function useSearchAddress(options: AddressSearchControllerOptions = {}) {
  const searchQuery = ref('')
  const suggestions = ref<PlaceSuggestion[]>([])
  const showSuggestions = ref(false)
  const searchingAddress = ref(false)
  const activeSuggestionIndex = ref(-1)
  const selectedPlace = ref<PlaceSuggestion>({} as PlaceSuggestion)

  const syncState = () => {
    searchQuery.value = ctrl.searchQuery
    suggestions.value = ctrl.suggestions
    showSuggestions.value = ctrl.showSuggestions
    searchingAddress.value = ctrl.searchingAddress
    activeSuggestionIndex.value = ctrl.activeSuggestionIndex
    selectedPlace.value = ctrl.selectedPlace
  }

  const ctrl = new AddressSearchController({
    ...options,
    onStateChange: () => {
      syncState()
      if (options.onStateChange) options.onStateChange()
    },
  })

  syncState()

  onBeforeUnmount(() => {
    ctrl.dispose()
  })

  return {
    searchQuery,
    suggestions,
    showSuggestions,
    searchingAddress,
    activeSuggestionIndex,
    selectedPlace,
    handleSearchInput: () => ctrl.handleSearchInput(searchQuery.value),
    clearSearch: () => ctrl.clearSearch(),
    selectDestination: (place: PlaceSuggestion) => ctrl.selectDestination(place),
    submitSearch: () => ctrl.submitSearch(),
    handleKeydown: (e: KeyboardEvent) => ctrl.handleKeydown(e),
  }
}
