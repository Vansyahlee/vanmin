import api from '@/services/api'

export const getSalesOrders = async () => {
    const response = await api.get(
        '/sales-orders',
    )

    return response.data
}

export const getSalesOrderByID = async id => {
    const response = await api.get(
        `/sales-orders/${id}`,
    )

    return response.data
}

export const createSalesOrder = async data => {
    const response = await api.post(
        '/sales-orders',
        data,
    )

    return response.data
}

export const confirmSalesOrder = async id => {
    const response = await api.patch(
        `/sales-orders/${id}/confirm`,
    )

    return response.data
}

export const startSalesOrderPicking = async id => {
    const response = await api.patch(
        `/sales-orders/${id}/picking`,
    )

    return response.data
}

export const finishSalesOrderPacking = async id => {
    const response = await api.patch(
        `/sales-orders/${id}/packed`,
    )

    return response.data
}

export const shipSalesOrder = async id => {
    const response = await api.patch(
        `/sales-orders/${id}/shipped`,
    )

    return response.data
}