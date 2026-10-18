import api from '@/services/api'

export const getTaxProfiles = async () => {
    const response = await api.get(
        '/tax-profiles',
    )

    return response.data
}

export const getCurrentTaxProfile = async () => {
    const response = await api.get(
        '/tax-profiles/current',
    )

    return response.data
}

export const saveStandardTaxProfile = async data => {
    const response = await api.post(
        '/tax-profiles/standard',
        data,
    )

    return response.data
}