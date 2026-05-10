type Resolve<T> = (value: T) => void;
type Reject = (reason: unknown) => void;

export class MiniPromise<T> {
  private readonly promise: Promise<T>;

  constructor(executor: (resolve: Resolve<T>, reject: Reject) => void) {
    this.promise = new Promise<T>(executor);
  }

  then<U>(onFulfilled: (value: T) => U | PromiseLike<U>): MiniPromise<U> {
    return new MiniPromise<U>((resolve, reject) => {
      this.promise.then((value) => Promise.resolve(onFulfilled(value)).then(resolve, reject), reject);
    });
  }

  catch<U>(onRejected: (reason: unknown) => U | PromiseLike<U>): MiniPromise<T | U> {
    return new MiniPromise<T | U>((resolve, reject) => {
      this.promise.then(resolve, (reason) => Promise.resolve(onRejected(reason)).then(resolve, reject));
    });
  }

  finally(onFinally: () => void): MiniPromise<T> {
    return new MiniPromise<T>((resolve, reject) => {
      this.promise.finally(onFinally).then(resolve, reject);
    });
  }

  await(): Promise<T> {
    return this.promise;
  }

  static resolve<T>(value: T): MiniPromise<T> {
    return new MiniPromise<T>((resolve) => resolve(value));
  }
}

async function main(): Promise<void> {
  const value = await MiniPromise.resolve(2).then((n) => n + 3).await();
  console.log(value);
}

if (require.main === module) void main();
