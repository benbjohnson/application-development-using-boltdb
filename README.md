Application Development Using BoltDB
====================================

### Abstract

We've been taught for decades that we need a complex database server to run our
applications. However, these servers incur a huge performance hit for most
queries and they are frequently misconfigured because of their operational
complexity which can cause slowness or downtime. In this talk, I'll show you how
to use a local, pure Go key/value store called BoltDB to build applications that
are both simple and fast. We will see how development and deployment become a
breeze once you ditch your complex database server.


### Target Audience

Go developers who are interested in simplifying their application development.


### Introduction

Software is too complex and too slow. We've seen the speed of CPUs increase
by orders of magnitude in the past few decades yet our applications seem to
require more hardware than ever. In the past several years I've used embedded
key/value databases for my applications because of their simplicity and speed.
Today I'm going to walk you through building a simple application using a 
pure Go key/value store I wrote called BoltDB.


### What is a embedded key/value database?

Before we dive in, let's talk about what an embedded key/value store even is!
"Embedded" refers to the database actually being compiled into your application
instead of a database server which you connect to over a socket. A good example
of an embedded database is SQLite.

However, SQLite is a relational database so let's talk about what "key/value"
means next. Key/value databases are extremely simple. They map a "key", which
is just a unique set of bytes, to a "value", which is an arbitrary set of bytes.
It helps to think of this in terms of relational databases. Your "key" would be
your primary key and your value would be the encoded row. In fact, most database
servers utilize a key/value database internally to store rows. Essentially,
the key/value database is just a persisted map.

Some key/value databases allow you to have multiple key/value mappings. In
BoltDB, these are called "buckets". Every key is unique in a bucket and points
to a value. Many times you can think of buckets like tables in a relational
database. You may have a "users" bucket or a "products" bucket.

Just as there are many database servers to choose from, there are also many
types of embedded key/value databases with different trade offs. Sometimes
you'll trade write performance for read performance or you'll trade
transactional safety for performance. For example, BoltDB is read optimized and
supports fully serializable ACID transactions. This makes it good for many read-
heavy applications that require strong guarantees.


### Getting started with BoltDB

One of the best things about using BoltDB is that installation process is so
simple. You don't need to install a server or even configure it. You just use
"go get" like any other Go package and it'll work on Windows, Mac, Linux,
Raspberry Pi, and even iOS and Android.

```
$ go get github.com/boltdb/bolt
```


#### Object encoding

One feature of relational databases that many of us take for granted is that
they handle encoding rows into bytes on disk. Since Bolt only works with byte
slices we'll need to handle that manually. Lucky for us, there are a LOT of
options for object serialization.

In Go, one of the most popular serialization libraries is Protocol Buffers
(also called "protobufs"). One implementation, called gogoprotobuf, is also
one of the fastest. With protobufs, we declare our serialization format and
then generate Go files for doing this quickly.

Let's take a look at an example application to see how we'd do this. This
application shows how to do CRUD for a simple "User" data store but I've also
built many other less traditional applications on Bolt such as message queues
and analytics.


#### Domain types

In our app, we have a single domain type called "User" with two fields: ID
& username. You can expand out to more types and nest objects but we'll stick
with a single object to keep things simple.

I like to separate out my domain types from my encoding types by placing my
encoding types in a subpackage called "internal". I do this for two reasons.
First, it keeps the generated protobufs code separate. And second, the
"internal" package is inaccessible from other packages outside our app and it's
hidden from godoc.

Inside our protobuf definition we can see that it matches our domain type with
a few exceptions. Since it's our binary representation we have to specify a size
for our integer type. Also, you'll notice numbers on the right. These are
essentially field IDs when it's encoded. When you add or remove fields you
don't need to do a migration like with a relational database. You simply add a
field with a higher number or delete a field.

Back in our store.go we can add code to generate our protobufs. This line
calls the protobuf compiler, "protoc", and will generate to
`internal/internal.pb.go`. If we look in there we can see it's a bunch of ugly
generated code.

Our domain type will convert to and from this protobuf type but we can hide it
all behind the `encoding.BinaryMarshaler` & `encoding.BinaryUnmarshaler`
interfaces. Our `MarshalBinary()` simply copies our fields in and marshals them
and the `UnmarshalBinary()` unmarshals the data and copies the fields out. This
is a bit more work than in relational databases but it's easy to write, test,
and migrate.


#### Initializing the store

Our `Store` type will be our application wrapper around our Bolt database. To
open a `bolt.DB`, we simply need to pass in the file path and the file
permissions to set if the file doesn't exist. This will use the umask so I
typically set my permissions to `0666` and let users set the umask to filter
that at runtime.

Once the database is open, we'll start a writable transaction. The `Begin()`
function is what starts a transaction. The `true` argument means that it's a
"writable" transaction. Bolt can have as many read transactions as you want but
only one write transaction gets processed at a time. This means it's important
to keep updates small and break up really large updates into smaller chunks. All
transactions operate under serializable isolation which means that all data will
be a snapshot of exactly how it was when the transaction started -- even if
other write transactions commit while the read transaction is in process.

The deferred rollback can look odd since we want to commit the transaction at
end. It's important in case you return an error early or your application
panics. All transactions need rollback or commit when they're done or else they
can block other operations later.

Within our transaction we call `CreateBucketIfNotExists()` to create our bucket
for our users. This is similar to a "CREATE TABLE IF NOT EXISTS" in SQL. If the
bucket doesn't exist then it's created. Otherwise it's ignored. Calling this
during initialization means that we won't have to check for it whenever we use
the "users" bucket. It's guaranteed to be there.

Finally, we commit our transaction and return the error, if any occurred while
saving. Bolt does not allow partial transactions so if a disk error occurs then
your entire transaction will be rolled back. The deferred rollback that we
called earlier will be ignored for this transaction since we have successfully
committed.

Closing the store is a simple task. Simply call `Close()` on the Bolt database
and it will release it's exclusive file lock and close the file descriptor.


#### Creating a user

Now that we have our database ready, let's create a user. In our `CreateUser()`
method, we'll start by creating a writable transaction just like we did before.
Then we'll grab the "Users" bucket from the transaction. We don't need to check
if it exists because we created it during initialization.

Next, we'll create a new ID for the user. Bolt has a nice feature called
sequences which are transactionally safe autoincrementing integers for each
bucket. Whenever we call `NextSequence()`, we'll get the bucket's next integer.
Once we grab it, we assign it to our user's ID.

Now our user is ready to be marshaled. We call `MarshalBinary()` and we get a
set of bytes which represents our encoded user. Easy peasy!

Since Bolt only works with bytes, we'll need to convert our ID to bytes. I
recommend using the `binary.BigEndian.PutUint64()` function for this. I use
big endian because it will sort our IDs in the right order.

[show big endian vs little endian on slide]

We'll use the bucket's `Put()` method to associate our encoded user with the
encoded ID. Then we'll commit and our data is saved.


#### Retrieving the user

Creating a user is just a matter of converting objects to bytes so retrieving a
user is simply converting bytes to objects. In our `User()` method we'll start
a transaction but this time we'll pass in `false` to specify a read-only
transaction. Again, read-only transactions can run completely in parallel so
this scales really well across multiple CPU cores.

Once we have our transaction, we call `Get()` on our bucket with an encoded
user ID and we get back the encoded user bytes. We can call `UnmarshalBinary()`
on a new `User` and decode the data. If the encoded bytes comes back as `nil`
then we know that the user doesn't exist and we can simply return a `nil` user.


#### Retrieving multiple users

Reading one user is good but many times we want to return a list of all users.
For this we'll need to use a Bolt cursor. A cursor is simply an object for
iterating over a bucket in order. It has a handful of methods we can use to
move forward, back or even jump around.

In our `Users()` method we'll grab a read-only transaction and a cursor from
our bucket. Then we'll iterate over every key/value in our bucket. We can
collapse it all into a simple "for" loop where we call `First()` at the
beginning and then `Next()` until we receive a `nil` key. For each value, we'll
unmarshal the user and add it to our slice.

If you need reverse sorting, you can call `Last()` and then `Prev()` on the
cursor. You can also use the `Seek()` method to jump to a specific spot. For
example, if we wanted to do pagination we could pass in an "options" object
into the method and have an offset and limit.


#### Updating a user

Now that we've created a user, let's update it. Let's look at the
`SetUsername()` method. This time we'll mix it up and use the `Update()` method
instead of `Begin()`. This method works just like `Begin(true)` except that
it executes a function in the context of the transaction. If the function
returns `nil` then the transaction commits. Otherwise if it returns an error or
panics then it will rollback the transaction.

First we'll retrieve our user by ID and unmarshal. In this case we're combining
the `Get()` and `UnmarshalBinary()` into a compound `if` block. I find it easier
to read if I group these related types of calls together. Next we simply update
the username on our user we just unmarshaled.

Now that we have our updated user object, we can simply remarshal it and
overwrite the previous value by calling `Put()` again.


#### Deleting a user

Finally, the last part of our CRUD store is the deletion. Delete is incredibly
simply. Simply call the `Delete()` method on the bucket. That's it!



### Bolt in Practice

That was the basics of doing CRUD operations with Bolt and we can talk about
more advanced use cases in a minute but first let's look at what running Bolt
in production looks like.

Internally, Bolt structures itself as a B+tree of pages which requires a lot of
random access at the file system so it's recommended that you run Bolt on an
SSD. Other embedded databases such as LevelDB are optimized for spinning disks.

Bolt also maps the database to memory using a read-only `mmap()` so byte slices
returned from buckets cannot be updated (or else it will SEGFAULT) and the
byte slices are only valid for the life of the transaction. The memory map
provides two amazing benefits. First, it means that data is never copied from
the database. You're accessing it directly from the operating system's page
cache. Second, since it's in the OS page cache, your hot data will persist in
memory across application restarts.


#### Backup & restore

From an operations standpoint, Bolt just uses a single file on disk so it's
simple to manage. However, as a library, there's not a standard CLI command to
backup your database but Bolt does provide a great option.

Transactions in Bolt implement the `io.WriterTo` interface which means they
can copy an entire snapshot of the database to an `io.Writer` with one line of
code. Depending on your application you may wish to provide an HTTP endpoint so
you can `curl` your backups or you can build an hourly snapshot to Amazon S3.

Another option in the works is a streaming transaction log so that you can
attach a process over the network to be an async replica. This is similar to
how Postgres replication works. This is still early in development though and
is not currently available.


### Performance

While performance is not a primary goal of Bolt, it is an important feature to
talk about. Bolt is read optimized so if your workload regularly consists of
tens of thousands of writes per second then you may want to look at write
optimized databases such as LevelDB.


#### Benchmarks

Benchmarks are not typically very useful but it's good to know a ballpark of
what a database can handle. I typically tell people the following on a machine
with an SSD. Expect to get up to 2,000 random writes/sec without optimization.
If you're bulk loading and sorting data then you can get up to 400K+ writes/sec.
Typically you're throttled by your hard drive speed.

On the read side, it depends on if your data is in the page cache. Typical CRUD
applications have maybe 10-20% of their data hot at any given time. That means
that if you have a 20GB database then 2 - 4GB is hot and that will be resident
in memory assuming you have that much RAM. For hot data, you can expect a read
transaction and a single bucket read to take a 1-2µs. If you're iterating over
hot data then you can see speeds of 20M keys/second. Again, all this data is
in the page cache and there's no data copy so it's really fast.

If your data is not hot then you'll be limited by the speed of your hard drive.
Expect reads of cold data on an SSD to take hundreds of microseconds or a few
milliseconds depending on the size of your database.


### Scaling with Bolt

One of the biggest criticisms of embedded databases is that they are embedded
and don't have a means of scaling or clustering. This is true, however, it's not
necessarily a reason to avoid embedded databases. There are several strategies
that can be used to obtain the safety required for your application while still
using an embedded database.


#### Vertical scaling

The easiest way to scale is still, by far, by scaling vertically. If your
bottleneck is with read queries then simply adding more CPU cores will scale
your Bolt application near linearly. This answer may sound too simplistic but
that's the beauty of it. It's really simple.


#### Horizontal scaling via sharding

Many applications -- especially SaaS applications -- can be partitioned by users
or accounts. This means that you can either assign each account to its own
database or use strategies like consistent hashing to group accounts into a
number of partitions. This will allow you to add additional machines and
rebalance the load of your application.


#### Data integrity in the face of catastrophic failure

Until streaming replication is ready for Bolt, windows of data loss are still
an issue that needs consideration when building with an embedded database. Many
cloud providers provide highly safe, redundant storage but even those can fail
every great once in a while.

For some applications, a daily backup may be reliable enough. For others, a
standby machine taking snapshots every 10 minutes may be required. Many
applications store critical data such as finances in a third party service like
Stripe so a 10 minute window may be acceptable.

Again, a Bolt database is simply a file and can be copied extremely quickly.
Expect a copy to take 3-5 seconds per gigabyte on an SSD.


### Other uses for Bolt beyond CRUD

TODO


### Conclusion

I believe that local key/value databases meet the requirements for many
applications while providing a simple, fast development experience. I have
shown you how to get up and running with Bolt and build a simple CRUD data
store. The code is not just straightforward but also incredibly fast and
can vertically scale for many workloads. Please consider using a local
key/value database in your next project!


