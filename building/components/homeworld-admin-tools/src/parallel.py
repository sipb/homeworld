import threading
import command


def parallel(func_a, func_b):
    no_result = object()
    results = [no_result, no_result]
    barrier = threading.Barrier(2)

    def main_a():
        barrier.wait()
        results[0] = func_a()

    def main_b():
        barrier.wait()
        results[1] = func_b()

    th1 = threading.Thread(target=main_a)
    th2 = threading.Thread(target=main_b)
    th1.start()
    th2.start()
    th1.join()
    th2.join()
    if results[0] is no_result:
        command.fail("no result received from thread A")
    if results[1] is no_result:
        command.fail("no result received from thread B")
    return results[0], results[1]
