type ListNodeI = {
    key: number
    value: number
    next: ListNode | null
    prev: ListNode | null
}

class ListNode {
    public key: number
    public value: number
    public next: ListNode | null
    public prev: ListNode | null

    constructor(key: number, value: number) {
        this.key = key
        this.value = value
        this.next = null
        this.prev = null
    }
}

class LRUCache {
    public size: number
    public capacity: number
    public map: Map<number, ListNode>
    public head: ListNode
    public tail: ListNode

    constructor(capacity: number) {
        this.map = new Map<number, ListNode>()
        this.capacity = capacity
        this.size = 0

        this.head = new ListNode(0, 0)
        this.tail = new ListNode(-1, -1)
        this.head.next = this.tail
        this.tail.prev = this.head
    }

    deleteNode(node: ListNode) {
        const prevNode = node.prev
        const nextNode = node.next
        if (!prevNode || !nextNode) return
        prevNode.next = nextNode
        nextNode.prev = prevNode
        node.prev = null 
        node.next = null 
    }

    addnode(node: ListNode) {
        const prevTail = this.tail.prev
        if (!prevTail) return
        prevTail.next = node
        node.prev = prevTail
        node.next = this.tail
        this.tail.prev = node
    }

    get(key: number) {
        const resultNode = this.map.get(key)
        if (!resultNode) return -1
        this.deleteNode(resultNode)
        this.addnode(resultNode)
        return resultNode.value
    }

    put(key: number, value: number) {
        const node = this.map.get(key)

        if (node) {
            node.value = value
            this.deleteNode(node)
            this.addnode(node)
            return
        }

        if (this.size === this.capacity) {
            const lruNode = this.head.next
            if (lruNode && lruNode !== this.tail) {
                this.deleteNode(lruNode)
                this.map.delete(lruNode.key) 
                this.size--
            }
        }

        const newNode = new ListNode(key, value)
        this.map.set(key, newNode)
        this.addnode(newNode)
        this.size++
    }
}

function main() {
    const lruCache = new LRUCache(2)
    lruCache.put(1, 1)
    lruCache.put(2, 2)
    console.log(lruCache.get(1))
    lruCache.put(3, 3)
    console.log(lruCache.get(2))
}

main()
