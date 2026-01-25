class ListNodee {
    public key: number
    public value: number
    public frequency: number
    public next: ListNodee | null
    public prev: ListNodee | null

    constructor(_key: number, _value: number) {
        this.key = _key
        this.value = _value
        this.frequency = 1
        this.next = null
        this.prev = null
    }
}

class DoublyLinkedList {
    public size: number
    public head: ListNodee
    public tail: ListNodee

    constructor() {
        this.size = 0
        this.head = new ListNodee(0, 0)
        this.tail = new ListNodee(0, 0)
        this.tail.prev = this.head
        this.head.next = this.tail
    }

    addNode(node: ListNodee) {
        const prevTail = this.tail.prev
        if (!prevTail) return
        prevTail.next = node
        node.prev = prevTail
        this.tail.prev = node
        node.next = this.tail
        this.size++
    }

    remove(node: ListNodee) {
        const prevNode = node.prev
        const nextNode = node.next
        if (!prevNode || !nextNode) return
        prevNode.next = nextNode
        nextNode.prev = prevNode
        this.size--
    }

    removeLast() {
        const headNext = this.head.next
        if (!headNext || headNext === this.tail) return null
        this.head.next = headNext.next
        headNext.next!.prev = this.head
        this.size--
        return headNext
    }
}

class LFUCache {
    public capacity: number
    public minFreq: number
    public keyToNode: Map<number, ListNodee>
    public freqToNodes: Map<number, DoublyLinkedList>

    constructor(capacity: number) {
        this.capacity = capacity
        this.minFreq = 0
        this.keyToNode = new Map()
        this.freqToNodes = new Map()
    }

    get(key: number) {
        const node = this.keyToNode.get(key)
        if (!node) return -1
        this.update(node)
        return node.value
    }

    update(node: ListNodee) {
        const list = this.freqToNodes.get(node.frequency)
        list?.remove(node)

        if (list?.size === 0 && this.minFreq === node.frequency) {
            this.minFreq++
        }

        node.frequency++
        let nextList = this.freqToNodes.get(node.frequency)
        if (!nextList) {
            nextList = new DoublyLinkedList()
            this.freqToNodes.set(node.frequency, nextList)
        }
        nextList.addNode(node)
    }

    put(key: number, value: number) {
        if (this.capacity === 0) return

        const node = this.keyToNode.get(key)
        if (node) {
            node.value = value
            this.update(node)
            return
        }

        if (this.keyToNode.size === this.capacity) {
            const list = this.freqToNodes.get(this.minFreq)!
            const evicted = list.removeLast()!
            this.keyToNode.delete(evicted.key)
        }

        const newNode = new ListNodee(key, value)
        this.keyToNode.set(key, newNode)

        let list = this.freqToNodes.get(1)
        if (!list) {
            list = new DoublyLinkedList()
            this.freqToNodes.set(1, list)
        }
        list.addNode(newNode)
        this.minFreq = 1
    }
}

function main2() {
    const cache = new LFUCache(2)

    cache.put(1, 10)
    cache.put(2, 20)

    console.log(cache.get(1)) // 10
    console.log(cache.get(1)) // 10

    cache.put(3, 30) // evicts key 2

    console.log(cache.get(2)) // -1
    console.log(cache.get(3)) // 30
    console.log(cache.get(1)) // 10

    cache.put(4, 40) // evicts key 3

    console.log(cache.get(3)) // -1
    console.log(cache.get(4)) // 40
    console.log(cache.get(1)) // 10
}


main2()
